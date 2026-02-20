package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
	"github.com/kiselevos/memento_game_bot/internal/models"
)

type SessionRepositoryInterface interface {
	Create(session *models.Session) (*models.Session, error)
	GetSessionByID(chatID int64) (*models.Session, error)
	ChangeIsActive(chatID int64) error
	AddUserToSession(session *models.Session, user *models.User) error
	AddPhotosCount(chatID int64) error
}

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
