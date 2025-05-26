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
	result := repo.DataBase.DB.Create(session)
	if result.Error != nil {
		return nil, result.Error
	}
	return session, nil
}

func (repo *SessionRepository) GetSessionByID(sessionID int64) (*models.Session, error) {

	var session models.Session
	result := repo.DataBase.DB.First(&session, "chat_id = ?", sessionID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &session, nil
}

// AddUserToSession - many to many table
func (repo *SessionRepository) AddUserToSession(session *models.Session, user *models.User) error {
	return repo.DataBase.Model(session).Association("Users").Append(user)
}

func (repo *SessionRepository) AddPhotosCount(session *models.Session) error {
	session.PhotosCount++
	result := repo.DataBase.Save(session)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
