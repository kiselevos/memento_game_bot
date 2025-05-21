package game

import "sync"

// GameSession - Хранит данные о конкретной партии игры
type GameSession struct {

	// Постоянные
	ChatID    int64           // Номер чата, где идет игра
	Score     map[int64]int   // Мапа с очками юзеров
	UsedTasks map[string]bool // Для отслеживаания используемых вопросов

	// Обнуляющиеся при новом раунде
	UserNames        map[int64]string //Список участников раунда (Для автоматичекого подсчета и окончания раунда?)
	FSM              *FSM             // Машина состояний
	Votes            map[int64]int64  // Кто кому отдал свой голос в раунде
	UsersPhoto       map[int64]string // Хранение фотографий, отпрвленных юзером
	CarrentTask      string           // Текущее задание
	IndexPhotoToUser map[int]int64    // Мапа для голосования(Индекс очердности фото к игроку)

	mu sync.Mutex
}

// GetUserName - возвращает имя или ник пользователя
func (s *GameSession) GetUserName(userID int64) string {
	if name, ok := s.UserNames[userID]; ok {
		return name
	}
	return "Анонимный Осётр"
}
