package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kiselevos/memento_game_bot/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Db struct {
	*gorm.DB
}

// NewDB - создание нового подключения к DB
func NewDB(conf *config.Config) (*Db, error) {
	var db *gorm.DB
	var err error

	dsn := conf.Db.Dsn
	maxAttempts := 5
	delay := 2 * time.Second

	for i := 1; i <= maxAttempts; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			PrepareStmt:            true,
			SkipDefaultTransaction: true,
			Logger:                 logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, err := db.DB()
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err = sqlDB.PingContext(ctx); err == nil {
					return &Db{db}, nil
				}
			}

			log.Println("Database connection failed, retrying...",
				"attempt", i, "error", err)
			time.Sleep(delay)

		}
	}

	return nil, fmt.Errorf("could not connect to database after %d attempts: %w", maxAttempts, err)
}
