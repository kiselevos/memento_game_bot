package game

import (
	"PhotoBattleBot/internal/models"
	"PhotoBattleBot/internal/repositories"
	"errors"
	"fmt"
	"log"
	"sync"

	"gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	sessions map[int64]*GameSession
	mu       sync.Mutex

	UserRepo    *repositories.UserRepository
	SessionRepo *repositories.SessionRepository
	TaskRepo    *repositories.TaskRepository
}

// NewGameManager создаёт и возвращает новый экземпляр GameManager
func NewGameManager(
	userRepo *repositories.UserRepository,
	sessionRepo *repositories.SessionRepository,
	taskRepo *repositories.TaskRepository) *GameManager {
	return &GameManager{
		sessions: make(map[int64]*GameSession),
		mu:       sync.Mutex{},

		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
		TaskRepo:    taskRepo,
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

	// Запись статистики в БД
	_, err := gm.SessionRepo.Create(&models.Session{ChatID: chatID})
	if err != nil {
		log.Printf("[ERROR] сессия %d не сохранена в базу данных %v", chatID, err)
	}

	return session
}

// StartNewRound - запускает новый раунд в текущей сессии
func (gm *GameManager) StartNewRound(session *GameSession, task string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Printf("[GAME] Новый раунд запущен в чате %d", session.ChatID)

	if !SafeTrigger(session.FSM, EventStartRound, "StartNewRound") {
		return fmt.Errorf("Ошибка перехода FSM")
	}

	session.CarrentTask = task
	session.UsedTasks[task] = true
	session.UsersPhoto = make(map[int64]string)

	return nil
}

func (gm *GameManager) TakePhoto(chatID int64, user *telebot.User, photoID string) {

	gm.mu.Lock()
	defer gm.mu.Unlock()

	session, _ := gm.sessions[chatID]

	s, err := gm.SessionRepo.GetSessionByID(chatID)
	if err != nil {
		log.Printf("[ERROR] Не удалось найти сессию %d: %v", chatID, err)
	}

	err = gm.SessionRepo.AddPhotosCount(s)
	if err != nil {
		log.Printf("[ERROR] Не удалось увеличить PhotosCount %d: %v", chatID, err)
	}

	if _, exist := session.UserNames[user.ID]; !exist {

		u, err := gm.UserRepo.GetUserByTGID(user.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			u = models.NewUser(user.ID, user.Username, user.FirstName)
			_, err = gm.UserRepo.Create(u)
			if err != nil {
				log.Printf("[ERROR] Не удалось создать пользователя %d: %v", user.ID, err)
			}
		}

		err = gm.SessionRepo.AddUserToSession(s, u)
		if err != nil {
			log.Printf("[ERROR] Не удалось привязать пользователя %d к сессии %d: %v", user.ID, chatID, err)
		}
	}

	session.TakePhoto(user, photoID)
}

func (gm *GameManager) StartVoting(session *GameSession) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Printf("[GAME] Голосование запущено в чате %d", session.ChatID)

	if !SafeTrigger(session.FSM, EventStartVote, "StartVoting") {
		return fmt.Errorf("Ошибка перехода FSM")
	}

	session.Votes = make(map[int64]int64)
	return nil
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
