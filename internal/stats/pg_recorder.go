package stats

import (
	"errors"
	"log"

	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/models"
	"github.com/kiselevos/memento_game_bot/internal/repositories"
	"gorm.io/gorm"
)

type PgRecorder struct {
	UserRepo    *repositories.UserRepository
	SessionRepo repositories.SessionRepositoryInterface
	TaskRepo    *repositories.TaskRepository
}

func NewPgRecorder(
	userRepo *repositories.UserRepository,
	sessionRepo repositories.SessionRepositoryInterface,
	taskRepo *repositories.TaskRepository,
) *PgRecorder {
	return &PgRecorder{UserRepo: userRepo, SessionRepo: sessionRepo, TaskRepo: taskRepo}
}

func (r *PgRecorder) CreateSessionRecord(chatID int64) {
	_, err := r.SessionRepo.Create(&models.Session{ChatID: chatID, IsActive: true})
	if err != nil {
		log.Printf("[DB ERROR] session %d not saved: %v", chatID, err)
	}
}

func (r *PgRecorder) IncrementTaskUsage(task string, hadPhotos bool) {
	if task == "" {
		return
	}
	if hadPhotos {
		if err := r.TaskRepo.AddUseCount(task); err != nil {
			log.Printf("[DB ERROR] add use_count task %q: %v", task, err)
		}
		return
	}
	if err := r.TaskRepo.AddSkipCount(task); err != nil {
		log.Printf("[DB ERROR] add skip_count task %q: %v", task, err)
	}
}

func (r *PgRecorder) RegisterUserLinkedToSession(chatID int64, user game.User) {
	u, err := r.UserRepo.GetUserByTGID(user.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = models.NewUser(user.ID, user.Username, user.FirstName)
		if _, err := r.UserRepo.Create(u); err != nil {
			log.Printf("[DB ERROR] create user %d: %v", user.ID, err)
		}
	}

	s, err := r.SessionRepo.GetSessionByID(chatID)
	if err != nil {
		log.Printf("[DB ERROR] get session chat=%d: %v", chatID, err)
		return
	}

	if err := r.SessionRepo.AddUserToSession(s, u); err != nil {
		log.Printf("[DB ERROR] link user %d to session %d: %v", user.ID, chatID, err)
	}

	if err := r.UserRepo.AddUserStatistic(user.ID, repositories.StatGame); err != nil {
		log.Printf("[DB ERROR] add StatGame to user %d: %v", user.ID, err)
	}
}

func (r *PgRecorder) IncrementPhotoSubmission(chatID int64, userID int64) {
	if err := r.SessionRepo.AddPhotosCount(chatID); err != nil {
		log.Printf("[DB ERROR] inc PhotosCount chat=%d: %v", chatID, err)
	}
	if err := r.UserRepo.AddUserStatistic(userID, repositories.StatPhoto); err != nil {
		log.Printf("[DB ERROR] add StatPhoto user=%d: %v", userID, err)
	}
}

func (r *PgRecorder) RecordVote(voterID int64) {
	if err := r.UserRepo.AddUserStatistic(voterID, repositories.StatVote); err != nil {
		log.Printf("[DB ERROR] add StatVote user=%d: %v", voterID, err)
	}
}

func (r *PgRecorder) IsFirstGame(chatID int64) (bool, error) {
	_, err := r.SessionRepo.GetSessionByID(chatID)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	return false, err
}

func (r *PgRecorder) RegisterRoundTask(chatID int64, task string) {
	if task == "" {
		return
	}

	_, err := r.TaskRepo.GetTaskByText(task)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t := models.NewTask(task)
		if _, err := r.TaskRepo.Create(t); err != nil {
			log.Printf("[DB ERROR] Не удалось добавить task %s: %v", task, err)
		}
	}
}
