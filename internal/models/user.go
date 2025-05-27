package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	TgUserId    int64  `gorm:"column:tg_user_id;uniqueIndex"`
	UserName    string `gorm:"column:username"`
	FirstName   string `gorm:"column:first_name"`
	GamesPlayed int    `gorm:"column:games_played"`
	PhotosSent  int    `gorm:"column:photos_sent"`
	UsersVote   int    `gorm:"column:users_vote"`
}

func NewUser(tgID int64, userName, firstName string) *User {
	return &User{
		TgUserId:  tgID,
		UserName:  userName,
		FirstName: firstName,
	}
}
