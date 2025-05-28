package models

import "gorm.io/gorm"

type Session struct {
	gorm.Model
	ChatID      int64   `gorm:"column:chat_id"`
	IsActive    bool    `gorm:"column:is_active"`
	Users       []*User `gorm:"many2many:session_users;"`
	PhotosCount int     `gorm:"column:photos_count"` // Количество фото, сыграных в сессии,
}

func NewSession(chatID int64) *Session {
	return &Session{
		ChatID: chatID,
	}
}
