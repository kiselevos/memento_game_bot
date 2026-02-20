package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
)

type FeedbackRepo struct {
	db *sql.DB
}

func NewFeedbackRepo(db *db.Db) *FeedbackRepo {
	return &FeedbackRepo{
		db: db.DB,
	}
}

func (repo *FeedbackRepo) Create(ctx context.Context, tgUserID int64, username, firstName, message string) error {
	_, err := repo.db.ExecContext(ctx, `
INSERT INTO feedback (tg_user_id, username, first_name, message, created_at)
VALUES ($1, $2, $3, $4, now())
`, tgUserID, nullifyEmpty(username), nullifyEmpty(firstName), message)
	if err != nil {
		return fmt.Errorf("feedback create: %w", err)
	}
	return nil
}
