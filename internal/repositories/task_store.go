package repositories

import (
	"context"

	"github.com/kiselevos/memento_game_bot/internal/models"
)

type TaskStore struct {
	taskRepo *TaskRepo
}

func NewTaskStore(tr *TaskRepo) *TaskStore {
	return &TaskStore{
		taskRepo: tr,
	}
}

func (ts *TaskStore) GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error) {
	return ts.taskRepo.GetActiveTaskList(ctx, category)
}
