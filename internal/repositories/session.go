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
