package repositories

import (
	"PhotoBattleBot/internal/models"
	"PhotoBattleBot/pkg/db"
)

type SessionRepository struct {
	DataBase *db.Db
}

func NewSessionRepository(db *db.Db) *SessionRepository {
	return &SessionRepository{
		DataBase: db,
	}
}

func (repo *SessionRepository) Create(session *models.Session) (*models.Session, error) {
	repo.DataBase.
		Model(&models.Session{}).
		Where("chat_id = ?", session.ChatID).
		Update("is_active", false)

	result := repo.DataBase.DB.Create(session)
	if result.Error != nil {
		return nil, result.Error
	}
	return session, nil
}

func (repo *SessionRepository) GetSessionByID(chatID int64) (*models.Session, error) {

	var session models.Session
	result := repo.DataBase.
		Where("chat_id = ? AND is_active = true", chatID).
		First(&session)
	if result.Error != nil {
		return nil, result.Error
	}
	return &session, nil
}

func (repo *SessionRepository) ChangeIsActive(chatID int64) error {
	session, err := repo.GetSessionByID(chatID)
	if err != nil {
		return err
	}
	session.IsActive = false
	result := repo.DataBase.Save(session)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// AddUserToSession - many to many table
func (repo *SessionRepository) AddUserToSession(session *models.Session, user *models.User) error {
	return repo.DataBase.Model(session).Association("Users").Append(user)
}

func (repo *SessionRepository) AddPhotosCount(chatID int64) error {

	session, err := repo.GetSessionByID(chatID)
	if err != nil {
		return err
	}
	session.PhotosCount++
	result := repo.DataBase.Save(session)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
