package repositories

import (
	"github.com/kiselevos/memento_game_bot/internal/models"
	"github.com/kiselevos/memento_game_bot/pkg/db"
)

const (
	StatGame  = "game"
	StatVote  = "vote"
	StatPhoto = "photo"
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

func (repo *UserRepository) GetUserByTGID(id int64) (*models.User, error) {

	var user models.User
	result := repo.DataBase.DB.First(&user, "tg_user_id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// AddUserStatistic - единая функция для увеличения показателей
func (repo *UserRepository) AddUserStatistic(userID int64, flag string) error {

	user, err := repo.GetUserByTGID(userID)
	if err != nil {
		return err
	}

	switch flag {
	case StatVote:
		user.UsersVote++
	case StatGame:
		user.GamesPlayed++
	case StatPhoto:
		user.PhotosSent++
	}

	result := repo.DataBase.Save(user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
