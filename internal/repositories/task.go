package repositories

import (
	"PhotoBattleBot/internal/models"
	"PhotoBattleBot/pkg/db"
)

type TaskRepository struct {
	DataBase *db.Db
}

func NewTaskRepository(db *db.Db) *UserRepository {
	return &UserRepository{
		DataBase: db,
	}
}

func (repo *TaskRepository) Create(task *models.Task) (*models.Task, error) {
	result := repo.DataBase.DB.Create(task)
	if result.Error != nil {
		return nil, result.Error
	}
	return task, nil
}
