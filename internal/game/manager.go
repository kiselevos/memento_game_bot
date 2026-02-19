package game

import (
	"fmt"
	"log"
	"sync"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/tasks"
)

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	actors map[int64]*chatActor
	mu     sync.Mutex

	stats StatsRecorder
}

// NewGameManager создаёт и возвращает новый экземпляр GameManager
func NewGameManager(stats StatsRecorder) *GameManager {
	if stats == nil {
		stats = NoopStatsRecorder{}
	}

	return &GameManager{
		actors: make(map[int64]*chatActor),
		mu:     sync.Mutex{},

		stats: stats,
	}
}

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

var (
	ErrNoSession      = fmt.Errorf("no active session")
	ErrNoTasksLeft    = fmt.Errorf("no tasks left")
	ErrRoundNotActive = fmt.Errorf("round not active")
)

// Чтобы не тянуть sessions в handlers
func (gm *GameManager) DoWithSession(chatID int64, fn func(s *GameSession) error) error {
	return gm.Do(chatID, func(a *chatActor) error {
		if a.session == nil {
			return ErrNoSession
		}
		return fn(a.session)
	})
}

// StartNewGameSession - запускает/перезапускает игру. Все очки стираются.
func (gm *GameManager) StartNewGameSession(chatID int64, user User) error {

	err := gm.Do(chatID, func(a *chatActor) error {
		log.Printf("[GAME] Игра запущена в чате %d", chatID)

		session := &GameSession{
			ChatID: chatID,
			FSM:    NewFSM(),
			Host:   user,

			Score:     make(map[int64]int),
			UsedTasks: make(map[string]bool),
			UserNames: make(map[int64]string),
		}

		session.UserNames[session.Host.ID] = DisplayNameHTML(&user)

		a.session = session
		return nil
	})

	if err != nil {
		return err
	}

	// Запись новой игровой сессии в БД
	gm.stats.CreateSessionRecord(chatID)
	return nil
}

// CheckFirstGame - Проверка на первую игру в группе.
func (gm *GameManager) CheckFirstGame(chatID int64) bool {
	first, err := gm.stats.IsFirstGame(chatID)
	if err != nil {
		return false
	}
	return first
}

// Обработка раунда через сессию и запись в DB
func (gm *GameManager) StartNewRound(chatID int64, tl *tasks.TasksList) (string, error) {

	var (
		task      string
		prevTask  string
		hadPhotos bool
	)

	err := gm.DoWithSession(chatID, func(s *GameSession) error {

		// Достаем новую task
		t, err := tl.GetRandomTask(s.UsedTasks)
		if err != nil {
			return ErrNoTasksLeft
		}
		task = t

		prevTask, hadPhotos, err = s.StartNewRound(task)
		return err
	})

	if err != nil {
		return "", err
	}

	// Запись таски в DB
	if prevTask != "" {
		gm.stats.IncrementTaskUsage(prevTask, hadPhotos)
	}
	gm.stats.RegisterRoundTask(chatID, task)

	return task, nil
}

func (gm *GameManager) SubmitPhoto(chatID int64, user *User, fileID string) (userName string, replaced bool, err error) {

	isNewUser := false

	err = gm.DoWithSession(chatID, func(s *GameSession) error {

		// Проверяем FSM
		if s.FSM.Current() != RoundStartState {
			return ErrRoundNotActive
		}

		// UsersPhoto nil
		if s.UsersPhoto == nil {
			fmt.Println("[ERROR] При запущенном раунде не создался объект UserPhoto")
		}

		// Проверка на нового User
		if _, ok := s.UserNames[user.ID]; !ok {
			isNewUser = true
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
		gm.stats.RegisterUserLinkedToSession(chatID, *user)
	}

	gm.stats.IncrementPhotoSubmission(chatID, user.ID)

	return userName, replaced, nil
}

// VOTING PROCESS
// VoteResult спец тип для ответов или CallBack или Messages
type VoteResult struct {
	Message    string
	IsCallback bool
	IsError    bool
}

func (gm *GameManager) StartVoting(chatID int64) ([]VotePhoto, error) {
	var photos []VotePhoto

	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		items, err := s.StartVoting()
		if err != nil {
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

	// DB — снаружи actor (и только если голос реально принят)
	if accepted {
		gm.stats.RecordVote(voter.ID)
		return &VoteResult{Message: msg, IsCallback: false}, nil
	}

	return &VoteResult{Message: msg, IsCallback: true}, nil
}

// Закончить голосование
func (gm *GameManager) FinishVoting(chatID int64) error {
	return gm.DoWithSession(chatID, func(s *GameSession) error {
		return s.FinishVoting()
	})
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
		log.Printf("[SaveSystemMsgID] BEFORE len=%d add=%d session_ptr=%p",
			len(s.SystemMsgIDs), msgID, s,
		)
		s.SystemMsgIDs = append(s.SystemMsgIDs, msgID)
		log.Printf("[SaveSystemMsgID] AFTER  len=%d session_ptr=%p",
			len(s.SystemMsgIDs), s,
		)
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

// Получить очки раунда
func (gm *GameManager) GetRoundScore(chatID int64) ([]PlayerScore, error) {
	var scores []PlayerScore

	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		scores = s.RoundScore()
		return nil
	})

	return scores, err
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

func (gm *GameManager) EndGame(chatID int64) error {
	return gm.Do(chatID, func(a *chatActor) error {
		a.session = nil
		return nil
	})
}
