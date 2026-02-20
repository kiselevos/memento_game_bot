package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
	"github.com/kiselevos/memento_game_bot/internal/models"
)

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *db.Db) *TaskRepo {
	return &TaskRepo{db: db.DB}
}

func (repo *TaskRepo) GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error) {

	const base = `SELECT id, text FROM tasks WHERE is_active = TRUE`

	var (
		rows *sql.Rows
		err  error
		out  []models.Task
	)

	if category != nil && *category != "" {
		rows, err = repo.db.QueryContext(ctx, base+` AND category = $1 ORDER BY id`, *category)
	} else {
		rows, err = repo.db.QueryContext(ctx, base+` ORDER BY id`)
	}
	if err != nil {
		return nil, fmt.Errorf("tasks list active: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t models.Task

		if err := rows.Scan(&t.ID, &t.Text); err != nil {
			return nil, fmt.Errorf("tasks scan: %w", err)
		}

		out = append(out, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tasks rows: %w", err)
	}

	return out, nil
}

func (repo *TaskRepo) IncUse(ctx context.Context, id, countPhoto int64) error {
	res, err := repo.db.ExecContext(ctx, `
UPDATE tasks
SET use_count = use_count + 1,
	photo_count = photo_count + $1,
    updated_at = now()
WHERE id = $2
`, countPhoto, id)
	if err != nil {
		return fmt.Errorf("tasks inc use: %w", err)
	}
	return ensureRowsAffected(res, fmt.Sprintf("tasks inc use: task id=%d not found", id))
}

func (repo *TaskRepo) IncSkip(ctx context.Context, id int64) error {
	res, err := repo.db.ExecContext(ctx, `
UPDATE tasks
SET skip_count = skip_count + 1,
    updated_at = now()
WHERE id = $1
`, id)
	if err != nil {
		return fmt.Errorf("tasks inc skip: %w", err)
	}
	return ensureRowsAffected(res, fmt.Sprintf("tasks inc skip: task id=%d not found", id))
}
