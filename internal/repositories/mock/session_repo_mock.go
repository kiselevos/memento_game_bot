package mock

import "github.com/kiselevos/memento_game_bot/internal/models"

// FakeSessionRepo — мок реализации SessionRepository
type FakeSessionRepo struct {
	Created []*models.Session
}

// Create сохраняет сессию в памяти (имитация вставки в БД)
func (f *FakeSessionRepo) Create(s *models.Session) (*models.Session, error) {
	f.Created = append(f.Created, s)
	return s, nil
}

func (f *FakeSessionRepo) GetSessionByID(chatID int64) (*models.Session, error)     { return nil, nil }
func (f *FakeSessionRepo) ChangeIsActive(chatID int64) error                        { return nil }
func (f *FakeSessionRepo) AddUserToSession(s *models.Session, u *models.User) error { return nil }
func (f *FakeSessionRepo) AddPhotosCount(chatID int64) error                        { return nil }
