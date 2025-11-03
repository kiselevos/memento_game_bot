package game

import (
	"errors"
	"fmt"
	"log"
	"sync"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/models"
	"github.com/kiselevos/memento_game_bot/internal/repositories"

	"gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

// GameManager - управляет активными игровыми сессиями
type GameManager struct {
	sessions map[int64]*GameSession
	mu       sync.Mutex

	UserRepo    *repositories.UserRepository
	SessionRepo repositories.SessionRepositoryInterface
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
	_, err := gm.SessionRepo.Create(&models.Session{ChatID: chatID, IsActive: true})
	if err != nil {
		log.Printf("[DB ERROR] сессия %d не сохранена в базу данных %v", chatID, err)
	}

	return session
}

// CheckFirstGame - Проверка на первую игру в группе.
func (gm *GameManager) CheckFirstGame(chatID int64) bool {

	_, err := gm.SessionRepo.GetSessionByID(chatID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	return false
}

// Сбор статистики task
func (gm *GameManager) saveTaskStats(session *GameSession) {
	prevTask := session.CarrentTask
	if prevTask == "" {
		return
	}
	if len(session.UsersPhoto) > 0 {
		err := gm.TaskRepo.AddUseCount(prevTask)
		if err != nil {
			log.Printf("[DB ERROR] Ошибка добавления use_count_task в базу данных %v", err)
		}
	} else {
		err := gm.TaskRepo.AddSkipCount(prevTask)
		if err != nil {
			log.Printf("[DB ERROR] Ошибка добавления skip_count_task в базу данных %v", err)
		}
	}
}

// StartNewRound - запускает новый раунд в текущей сессии
func (gm *GameManager) StartNewRound(session *GameSession, task string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.saveTaskStats(session)

	log.Printf("[GAME] Новый раунд запущен в чате %d", session.ChatID)

	if !SafeTrigger(session.FSM, EventStartRound, "StartNewRound") {
		return fmt.Errorf("Ошибка перехода FSM")
	}

	t, err := gm.TaskRepo.GetTaskByText(task)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t = models.NewTask(task)
		_, err = gm.TaskRepo.Create(t)
		if err != nil {
			log.Printf("[DB ERROR] Не удалось добавить task %s: %v", task, err)
		}
	}

	session.CarrentTask = task
	session.UsedTasks[task] = true
	session.UsersPhoto = make(map[int64]string)

	return nil
}

func (gm *GameManager) addSessionUserIfNotExist(session *GameSession, user *telebot.User) {
	if _, exist := session.UserNames[user.ID]; exist {
		return
	}

	chatID := session.ChatID
	userID := user.ID

	u, err := gm.UserRepo.GetUserByTGID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = models.NewUser(userID, user.Username, user.FirstName)
		_, err = gm.UserRepo.Create(u)
		if err != nil {
			log.Printf("[DB ERROR] Не удалось создать пользователя %d: %v", userID, err)
		}
	}

	s, err := gm.SessionRepo.GetSessionByID(chatID)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось найти сессию в БД. Чат %d: %v", chatID, err)
		return
	}

	err = gm.SessionRepo.AddUserToSession(s, u)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось привязать пользователя %d к сессии %d: %v", userID, chatID, err)
	}

	err = gm.UserRepo.AddUserStatistic(user.ID, repositories.StatGame)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось добавить игру участнику %d в сессии %d: %v", userID, chatID, err)
	}

}

func (gm *GameManager) TakePhoto(chatID int64, user *telebot.User, photoID string) {

	gm.mu.Lock()
	defer gm.mu.Unlock()

	session, _ := gm.sessions[chatID]

	gm.addSessionUserIfNotExist(session, user)

	err := gm.SessionRepo.AddPhotosCount(chatID)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось увеличить PhotosCount %d: %v", chatID, err)
	}

	err = gm.UserRepo.AddUserStatistic(user.ID, repositories.StatPhoto)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось добавить фото участнику %d в сессии %d: %v", user.ID, chatID, err)
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

// VoteResult спец тип для ответов или CallBack или Messages
type VoteResult struct {
	Message    string
	IsCallback bool
	IsError    bool
}

func (gm *GameManager) RegisterVote(chatID int64, voter *telebot.User, photoNum int) (*VoteResult, error) {

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

	// if targetUserID == voter.ID {
	// return &VoteResult{
	// 		Message:    messages.VotedForSelf,
	// 		IsCallback: true,
	// 	}, nil
	// }

	session.Votes[voter.ID] = targetUserID
	session.Score[targetUserID]++

	// Запись статистики
	err := gm.UserRepo.AddUserStatistic(voter.ID, repositories.StatVote)
	if err != nil {
		log.Printf("[DB ERROR] Не удалось добавить голос для %d: %v", voter.ID, err)
	}

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
