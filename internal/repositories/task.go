package repositories

import (
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
	"github.com/kiselevos/memento_game_bot/internal/models"

	"gorm.io/gorm"
)

type TaskRepository struct {
	DataBase *db.Db
}

func NewTaskRepository(db *db.Db) *TaskRepository {
	return &TaskRepository{
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

func (repo *TaskRepository) GetTaskByText(text string) (*models.Task, error) {

	var task models.Task
	result := repo.DataBase.DB.First(&task, "text = ?", text)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

func (repo *TaskRepository) AddUseCount(text string) error {

	result := repo.DataBase.
		Model(&models.Task{}).
		Where("text = ?", text).
		UpdateColumn("use_count", gorm.Expr("use_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("Ошибка уввеличения use_count: %v", result.Error)
	}

	return nil
}

func (repo *TaskRepository) AddSkipCount(text string) error {
	result := repo.DataBase.
		Model(&models.Task{}).
		Where("text = ?", text).
		UpdateColumn("skip_count", gorm.Expr("skip_count + ?", 1))

	if result.Error != nil {
		return fmt.Errorf("Ошибка уввеличения skip_count: %v", result.Error)
	}

	return nil
}
