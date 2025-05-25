package models

import "gorm.io/gorm"

type Session struct {
	gorm.Model
	ChatID      uint64
	Users       []*User
	RoundsCount int // Количество сыграных раундов (добавлена хоть одна фотография)
}

func NewSession(chatID uint64) *Session {
	return &Session{
		ChatID: chatID,
	}
}

func (s *Session) AddUserToSeeeion(user *User) {
	s.Users = append(s.Users, user)
}
