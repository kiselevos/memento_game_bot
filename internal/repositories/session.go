package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *db.Db) *SessionRepo {
	return &SessionRepo{
		db: db.DB,
	}
}

// Добавляем сессию
func (repo *SessionRepo) CreateSession(ctx context.Context, chatID int64) (int64, error) {

	var id int64

	err := repo.db.QueryRowContext(ctx, `
INSERT INTO sessions (chat_id, started_at)
VALUES ($1, now())
RETURNING id
`, chatID).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("sessions create: %w", err)
	}

	return id, nil
}

// Завершение сессии
func (repo *SessionRepo) FinishSession(ctx context.Context, sessionID int64) error {
	_, err := repo.db.ExecContext(ctx, `
UPDATE sessions
SET ended_at = now()
WHERE id = $1
`, sessionID)

	if err != nil {
		return fmt.Errorf("sessions end: %w", err)
	}

	return nil
}

// Проверка на наличие игор в чате
func (repo *SessionRepo) HasAnySession(ctx context.Context, chatID int64) (bool, error) {
	var exists bool
	err := repo.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM sessions
  WHERE chat_id = $1
  LIMIT 1
)
`, chatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sessions exists: %w", err)
	}
	return exists, nil
}

// Добавить юзера к сессии
func (repo *SessionRepo) IncPlayers(ctx context.Context, sessionID int64) error {
	if sessionID == 0 {
		return nil
	}
	_, err := repo.db.ExecContext(ctx, `
        UPDATE sessions
        SET players = players + 1
        WHERE id = $1
    `, sessionID)
	if err != nil {
		return fmt.Errorf("inc session players: %w", err)
	}
	return nil
}
