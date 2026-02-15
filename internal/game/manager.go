package game

import (
	"fmt"
	"log"
	"sync"

	messages "github.com/kiselevos/memento_game_bot/assets"
)

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	sessions map[int64]*GameSession
	mu       sync.Mutex

	stats StatsRecorder
}

// NewGameManager создаёт и возвращает новый экземпляр GameManager
func NewGameManager(stats StatsRecorder) *GameManager {
	if stats == nil {
		stats = NoopStatsRecorder{}
	}

	return &GameManager{
		sessions: make(map[int64]*GameSession),
		mu:       sync.Mutex{},

		stats: stats,
	}
}

// GetSession возвращает GameSession по chatID и bool
func (gm *GameManager) GetSession(chatID int64) (*GameSession, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	session, ok := gm.sessions[chatID]
	return session, ok
}

// StartNewGameSession - запускает/перезапускает игру. Все очки стираются.
func (gm *GameManager) StartNewGameSession(chatID int64) *GameSession {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Printf("[GAME] Игра запущена в чате %d", chatID)

	session := &GameSession{
		ChatID: chatID,
		FSM:    NewFSM(),

		Score:     make(map[int64]int),
		UsedTasks: make(map[string]bool),
		UserNames: make(map[int64]string),

		mu: sync.Mutex{},
	}

	gm.sessions[chatID] = session

	// Запись новой игровой сессии в БД
	gm.stats.CreateSessionRecord(chatID)

	return session
}

// CheckFirstGame - Проверка на первую игру в группе.
func (gm *GameManager) CheckFirstGame(chatID int64) bool {
	first, err := gm.stats.IsFirstGame(chatID)
	if err != nil {
		return false
	}
	return first
}

// Сбор статистики task
func (gm *GameManager) saveTaskStats(session *GameSession) {
	prevTask := session.CarrentTask
	if prevTask == "" {
		return
	}

	gm.stats.IncrementTaskUsage(prevTask, len(session.UsersPhoto) > 0)
}

// StartNewRound - запускает новый раунд в текущей сессии
func (gm *GameManager) StartNewRound(session *GameSession, task string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.saveTaskStats(session)

	log.Printf("[GAME] Новый раунд запущен в чате %d", session.ChatID)

	if !SafeTrigger(session.FSM, EventStartRound, "StartNewRound") {
		return fmt.Errorf("oшибка перехода FSM")
	}

	// Запись таски в DB
	gm.stats.RegisterRoundTask(session.ChatID, task)

	session.CarrentTask = task
	session.UsedTasks[task] = true
	session.UsersPhoto = make(map[int64]string)

	return nil
}

func (gm *GameManager) addSessionUserIfNotExist(session *GameSession, user *User) {

	if _, exist := session.UserNames[user.ID]; exist {
		return
	}

	gm.stats.RegisterUserLinkedToSession(session.ChatID, *user)
}

func (gm *GameManager) TakePhoto(chatID int64, user *User, photoID string) {

	gm.mu.Lock()
	defer gm.mu.Unlock()

	session := gm.sessions[chatID]

	gm.addSessionUserIfNotExist(session, user)

	// Запись статистики в DB
	gm.stats.IncrementPhotoSubmission(chatID, user.ID)

	session.TakePhoto(user, photoID)
}

func (gm *GameManager) StartVoting(session *GameSession) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Printf("[GAME] Голосование запущено в чате %d", session.ChatID)

	if !SafeTrigger(session.FSM, EventStartVote, "StartVoting") {
		return fmt.Errorf("oшибка перехода FSM")
	}

	session.Votes = make(map[int64]int64)
	return nil
}

// VoteResult спец тип для ответов или CallBack или Messages
type VoteResult struct {
	Message    string
	IsCallback bool
	IsError    bool
}

func (gm *GameManager) RegisterVote(chatID int64, voter *User, photoNum int) (*VoteResult, error) {

	gm.mu.Lock()
	defer gm.mu.Unlock()

	session, exist := gm.sessions[chatID]
	if !exist || session.FSM.Current() != VoteState {
		return &VoteResult{
			Message:    messages.VotedEarler,
			IsCallback: true,
		}, nil
	}

	if _, voted := session.Votes[voter.ID]; voted {
		return &VoteResult{
			Message:    messages.VotedAlready,
			IsCallback: true,
		}, nil
	}

	targetUserID, ok := session.IndexPhotoToUser[photoNum]
	if !ok {
		log.Printf("[ERROR] Неизвестный номер фото %d в чате %d", photoNum, chatID)
		return &VoteResult{
			Message:    messages.ErrorMessagesForUser,
			IsCallback: true,
			IsError:    true,
		}, fmt.Errorf("unknown photo")
	}

	// Голосование за себя
	if targetUserID == voter.ID {
		return &VoteResult{
			Message:    messages.VotedForSelf,
			IsCallback: true,
		}, nil
	}

	session.Votes[voter.ID] = targetUserID
	session.Score[targetUserID]++

	// Запись статистики голосования
	gm.stats.RecordVote(voter.ID)

	return &VoteResult{
		Message:    fmt.Sprintf("%s проголосовал(а)", session.GetUserName(voter.ID)),
		IsCallback: false,
	}, nil
}

func (gm *GameManager) FinishVoting(session *GameSession) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	SafeTrigger(session.FSM, EventFinishVote, "FinishVoting")
}

func (gm *GameManager) EndGame(chatID int64) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	delete(gm.sessions, chatID)
}
