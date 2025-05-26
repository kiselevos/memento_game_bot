package models

import "gorm.io/gorm"

type Session struct {
	gorm.Model
	ChatID      int64   `gorm:"column:chat_id"`
	Users       []*User `gorm:"many2many:session_users;"`
	RoundsCount int     `gorm:"column:rounds_count"` // Количество сыграных раундов (добавлена хоть одна фотография)
}

func NewSession(chatID int64) *Session {
	return &Session{
		ChatID: chatID,
	}
}
