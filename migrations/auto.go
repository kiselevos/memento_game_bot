package main

import (
	"log"

	"github.com/kiselevos/memento_game_bot/config"
	"github.com/kiselevos/memento_game_bot/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	dsn := config.GetDsn()
	log.Println("DSN used in Migrate:", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	err = db.AutoMigrate(&models.User{}, &models.Session{}, &models.Task{})

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations applied successfully")
}
