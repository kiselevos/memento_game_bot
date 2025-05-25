package repositories

import (
	"PhotoBattleBot/internal/models"
	"PhotoBattleBot/pkg/db"
)

type UserRepository struct {
	DataBase *db.Db
}

func NewUserRepository(db *db.Db) *UserRepository {
	return &UserRepository{
		DataBase: db,
	}
}

func (repo *UserRepository) Create(user *models.User) (*models.User, error) {
	result := repo.DataBase.DB.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func (repo *UserRepository) GetUserByTGID(id uint64) (*models.User, error) {

	var user models.User
	result := repo.DataBase.DB.First(&user, "TgUserId = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}
