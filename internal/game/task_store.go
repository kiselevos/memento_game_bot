package game

import (
	"context"

	"github.com/kiselevos/memento_game_bot/internal/models"
)

type TaskStore interface {
	GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error)
}

type NoopTaskStore struct{}

func (NoopTaskStore) GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error) {
	return []models.Task{}, nil
}
