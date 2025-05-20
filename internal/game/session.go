package game

import "sync"

// Тип для орисания состояния игры
type RoundState int

// Constants for game state
const (
	WaitingState RoundState = iota
	RoundStartState
	VoteState
)

// GameSession - Хранит данные о конкретной партии игры
type GameSession struct {

	// Постоянные
	ChatID    int64           // Номер чата, где идет игра
	Score     map[int64]int   // Мапа с очками юзеров
	UsedTasks map[string]bool // Для отслеживаания используемых вопросов

	// Обнуляющиеся при новом раунде
	UserNames   map[int64]string //Список участников раунда (Для автоматичекого подсчета и окончания раунда?)
	State       RoundState       // Текущее состояне игры
	Votes       map[int64]int64  // Кто кому отдал свой голос в раунде
	UsersPhoto  map[int64]string // Хранение фотографий, отпрвленных юзером
	CarrentTask string           // Текущее задание

	mu *sync.Mutex
}

// GetUserName - возвращает имя или ник пользователя
func (s *GameSession) GetUserName(userID int64) string {
	if name, ok := s.UserNames[userID]; ok {
		return name
	}
	return "Анонимный Осётр"
}
