package main

import (
	"PhotoBattleBot/config"
	"PhotoBattleBot/internal/models"
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file", err)
	}

	dsn := config.GetDsn()
	println("DSN used:", dsn) // Добавил вывод
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&models.User{}, &models.Session{}, &models.Task{})
}
