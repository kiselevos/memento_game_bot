package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	TgUserId  uint64 `gorm:"uniqueIndex"`
	UserName  string
	FirstName string
}

func NewUser(tgID uint64, userName, firstName string) *User {
	return &User{
		TgUserId:  tgID,
		UserName:  userName,
		FirstName: firstName,
	}
}
