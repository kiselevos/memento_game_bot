package game

import "sync"

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	sessions map[int64]*GameSession
	mu       *sync.Mutex
}

// NewGameManager создаёт и возвращает новый экземпляр GameManager
func NewGameManager() *GameManager {
	return &GameManager{
		sessions: make(map[int64]*GameSession),
		mu:       &sync.Mutex{},
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

	session := &GameSession{
		ChatID: chatID,
		State:  WaitingState,

		Score:     make(map[int64]int),
		UsedTasks: make(map[string]bool),
	}

	gm.sessions[chatID] = session

	return session
}

// StartNewRound - запускает новый раунд в текущей сессии
func (gm *GameManager) StartNewRound(session *GameSession, task string) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	session.CarrentTask = task
	session.State = RoundStartState
	session.UsedTasks[task] = true
	session.UsersPhoto = make(map[int64]string) // игроки -> фото
	session.UserNames = make(map[int]string)    // Имена игроков
	session.Votes = make(map[int64]int64)       // Голосование

}
