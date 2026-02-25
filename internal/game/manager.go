package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/logging"
	"github.com/kiselevos/memento_game_bot/internal/models"
	"github.com/kiselevos/memento_game_bot/internal/tasks"
)

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	appCtx context.Context
	actors map[int64]*chatActor
	mu     sync.Mutex

	stats StatsRecorder
	tasks TaskStore
}

// NewGameManager создаёт и возвращает новый экземпляр GameManager
func NewGameManager(ctx context.Context, stats StatsRecorder, tasks TaskStore) *GameManager {
	if stats == nil {
		stats = NoopStatsRecorder{}
	}

	if tasks == nil {
		tasks = NoopTaskStore{}
	}

	return &GameManager{
		appCtx: ctx,
		actors: make(map[int64]*chatActor),
		mu:     sync.Mutex{},

		stats: stats,
		tasks: tasks,
	}
}

var (
	ErrNoSession          = fmt.Errorf("game: no active session")
	ErrNoTasksLeft        = fmt.Errorf("game: no tasks left")
	ErrRoundNotActive     = fmt.Errorf("game: round not active")
	ErrWrongState         = fmt.Errorf("game: wrong game state")
	ErrInvariantViolation = fmt.Errorf("game: invariant violation")
	ErrLoadTasksFromDB    = fmt.Errorf("game: failed to load tasks from db")
	ErrLoadTasksLocal     = fmt.Errorf("game: local failed to load tasks")
	ErrOnlyHost           = fmt.Errorf("game: not host")
)

// Механизм очереди во избежание data race
func (gm *GameManager) Do(chatID int64, fn func(a *chatActor) error) error {
	gm.mu.Lock()
	a, ok := gm.actors[chatID]
	if !ok {
		a = newChatActor(chatID)
		gm.actors[chatID] = a
	}
	gm.mu.Unlock()

	reply := make(chan error, 1)
	a.inbox <- actorMsg{fn: fn, reply: reply}
	return <-reply
}

// Чтобы не тянуть sessions в handlers
func (gm *GameManager) DoWithSession(chatID int64, fn func(s *GameSession) error) error {
	return gm.Do(chatID, func(a *chatActor) error {
		if a.session == nil {
			return ErrNoSession
		}
		return fn(a.session)
	})
}

func (gm *GameManager) DoAsHost(
	chatID int64,
	userID int64,
	fn func(s *GameSession) error,
) error {
	return gm.Do(chatID, func(a *chatActor) error {
		if a.session == nil {
			return ErrNoSession
		}

		if a.session.Host.ID != userID {
			return ErrOnlyHost
		}

		return fn(a.session)
	})
}

// StartNewGameSession - запускает/перезапускает игру. Все очки стираются.
func (gm *GameManager) StartNewGameSession(chatID int64, user User) (bool, error) {

	fallbackUsed := false

	// Достаем вопросы для игровой сессии
	dbCtx, cancel := gm.dbCtx()
	taskList, err := gm.tasks.GetActiveTaskList(dbCtx, nil)
	cancel()

	if err != nil || len(taskList) == 0 {
		taskList, err = tasks.LoadTasksFromFile()
		if err != nil {
			return fallbackUsed, fmt.Errorf("%w: %w", ErrLoadTasksLocal, err)
		}
		fallbackUsed = true
	}

	tasks.ShuffleTasks(taskList)

	// Запись новой игровой сессии в БД
	dbCtx, cancel = gm.dbCtx()
	sessionID := gm.stats.CreateSessionRecord(dbCtx, chatID)
	cancel()

	err = gm.Do(chatID, func(a *chatActor) error {

		session := &GameSession{
			SessionID:   sessionID,
			ChatID:      chatID,
			FSM:         NewFSM(),
			Host:        user,
			CountRounds: 1,
			CarrentTask: models.Task{},

			Score:     make(map[int64]int),
			UserNames: make(map[int64]string),
			Tasks:     taskList,
		}

		a.session = session
		return nil
	})

	if err != nil {
		return fallbackUsed, err
	}

	return fallbackUsed, nil
}

// CheckFirstGame - Проверка на первую игру в группе.
func (gm *GameManager) CheckFirstGame(chatID int64) bool {
	dbCtx, cancel := gm.dbCtx()
	defer cancel()

	return gm.stats.IsFirstGame(dbCtx, chatID)
}

// Обработка раунда через сессию и запись в DB
func (gm *GameManager) StartNewRound(chatID, userID int64) (int, string, error) {

	var (
		round      int
		newTask    models.Task
		prevTaskID int64
		countPhoto int
	)

	err := gm.DoAsHost(chatID, userID, func(s *GameSession) error {
		// Достаем текущий раунд
		round = s.CountRounds

		var err error
		prevTaskID, countPhoto, newTask, err = s.StartNewRound()

		if errors.Is(err, ErrInvalidTransition) {
			return fmt.Errorf("%w: %v", ErrWrongState, err)
		}

		return err
	})

	if err != nil {
		return 0, "", err
	}

	// Запись таски в DB
	if prevTaskID != 0 && isGroupChat(chatID) {
		dbCtx, cancel := gm.dbCtx()
		gm.stats.IncrementTaskUsage(dbCtx, prevTaskID, int64(countPhoto))
		cancel()
	}

	return round, newTask.Text, nil
}

func (gm *GameManager) SubmitPhoto(chatID int64, user *User, fileID string) (userName string, replaced bool, err error) {

	var (
		sessionID int64
		isNewUser bool
	)

	err = gm.DoWithSession(chatID, func(s *GameSession) error {

		// Проверяем FSM
		if s.FSM.Current() != RoundStartState {
			return ErrRoundNotActive
		}

		// UsersPhoto nil
		if s.UsersPhoto == nil {
			slog.Default().Error("invariant violated: UsersPhoto is nil",
				"chat_id", chatID,
				"user_id", user.ID,
				"action", "submit_photo",
				"state", s.FSM.Current(),
			)
			logging.Notify(slog.LevelError, "invariant violated: UsersPhoto is nil",
				"chat_id", chatID,
				"user_id", user.ID,
				"state", s.FSM.Current(),
			)
			return ErrInvariantViolation
		}

		// Проверка на нового User
		if _, ok := s.UserNames[user.ID]; !ok {
			isNewUser = true
			sessionID = s.SessionID
		}

		replaced = s.TakePhoto(user, fileID)
		userName = s.GetUserName(user.ID)
		return nil
	})

	if err != nil {
		return
	}

	// Запись статистики в DB
	if isNewUser {
		dbCtx, cancel := gm.dbCtx()
		gm.stats.StatsForStartNewGame(dbCtx, *user, sessionID)
		cancel()
	}

	return userName, replaced, nil
}

// VOTING PROCESS
// VoteResult спец тип для ответов или CallBack или Messages
type VoteResult struct {
	Message    string
	IsCallback bool
	IsError    bool
}

func (gm *GameManager) StartVoting(chatID, userID int64) ([]VotePhoto, error) {
	var photos []VotePhoto

	err := gm.DoAsHost(chatID, userID, func(s *GameSession) error {
		items, err := s.StartVoting()
		if err != nil {
			if errors.Is(err, ErrInvalidTransition) {
				return fmt.Errorf("%w: %v", ErrWrongState, err)
			}
			return err
		}

		photos = items
		return nil
	})

	return photos, err
}

func (gm *GameManager) RegisterVote(chatID int64, voter *User, photoNum int) (*VoteResult, error) {

	var (
		accepted bool
		msg      string
		resErr   error
	)

	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		accepted, msg, resErr = s.RegisterVote(voter, photoNum)
		return resErr
	})

	if err == ErrNoSession {
		return &VoteResult{Message: messages.GameNotStarted, IsCallback: true, IsError: true}, nil
	}
	if err != nil {
		return &VoteResult{Message: messages.ErrorMessagesForUser, IsCallback: true, IsError: true}, err
	}

	if accepted {
		return &VoteResult{Message: msg, IsCallback: false}, nil
	}

	return &VoteResult{Message: msg, IsCallback: true}, nil
}

// Закончить голосование и получить очки
func (gm *GameManager) FinishVoting(chatID, userID int64) ([]PlayerScore, error) {

	var scores []PlayerScore
	var usersIDs []int64

	err := gm.DoAsHost(chatID, userID, func(s *GameSession) error {
		if err := s.FinishVoting(); err != nil {
			if errors.Is(err, ErrInvalidTransition) {
				return fmt.Errorf("%w: %v", ErrWrongState, err)
			}
			return err
		}
		usersIDs = s.GetPlayersIDs()
		scores = s.RoundScore()
		return nil
	})

	if err == nil {
		dbCtx, cancel := gm.dbCtx()
		gm.stats.UsersVotesStatsUpdate(dbCtx, scores, usersIDs)
		cancel()
	}

	return scores, err
}

// Сохранеям в сессии фото msg_id для последующего удаления
func (gm *GameManager) SaveVotePhotoMsgID(chatID int64, photoNum int, msgID int) error {
	return gm.DoWithSession(chatID, func(s *GameSession) error {
		s.VotePhotoMsgIDs[photoNum] = msgID
		return nil
	})
}

// Сохранеям в сессии системные msg_id для последующего удаления
func (gm *GameManager) SaveSystemMsgID(chatID int64, msgID int) error {
	return gm.DoWithSession(chatID, func(s *GameSession) error {

		s.SystemMsgIDs = append(s.SystemMsgIDs, msgID)
		return nil
	})
}

type CleanupIDs struct {
	VotePhotoMsgIDs []int
	SystemMsgIDs    []int
}

// Получаем photo_id для удаления
func (gm *GameManager) PopMsgIDs(chatID int64) (CleanupIDs, error) {
	var out CleanupIDs

	err := gm.DoWithSession(chatID, func(s *GameSession) error {

		for _, msgID := range s.VotePhotoMsgIDs {
			out.VotePhotoMsgIDs = append(out.VotePhotoMsgIDs, msgID)
		}
		s.VotePhotoMsgIDs = nil

		for _, msgID := range s.SystemMsgIDs {
			out.SystemMsgIDs = append(out.SystemMsgIDs, msgID)
		}
		s.SystemMsgIDs = nil
		return nil
	})
	return out, err
}

// Получить финальный счет игры
func (gm *GameManager) GetTotalScore(chatID int64) ([]PlayerScore, error) {
	var scores []PlayerScore

	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		scores = s.TotalScore()
		return nil
	})

	return scores, err
}

func (gm *GameManager) EndGame(chatID, userID int64) error {

	var sessionID int64

	err := gm.Do(chatID, func(a *chatActor) error {

		if a.session == nil {
			return ErrNoSession
		}

		if a.session.Host.ID != userID {
			return ErrOnlyHost
		}

		sessionID = a.session.SessionID

		a.session = nil
		return nil
	})

	if err != nil {
		return err
	}

	if sessionID != 0 {

		dbCtx, cancel := gm.dbCtx()
		gm.stats.FinishSessionRecord(dbCtx, sessionID)
		cancel()
	}

	return nil
}

func (gm *GameManager) dbCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(gm.appCtx, 5*time.Second)
}

// Проверка на групповой чат
func isGroupChat(chatID int64) bool {
	return chatID < 0
}
