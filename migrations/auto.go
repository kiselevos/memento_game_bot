package main

import (
	"PhotoBattleBot/config"
	"PhotoBattleBot/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(postgres.Open(config.GetDsn()), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&models.User{}, &models.Session{}, &models.Task{})
}
