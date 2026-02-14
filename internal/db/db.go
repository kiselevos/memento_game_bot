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
func NewDB(conf *config.DbConfig) (*Db, error) {
	var err error

	for i := 1; i <= conf.MaxAttempts; i++ {
		db, err := gorm.Open(postgres.Open(conf.Dsn), &gorm.Config{
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
			time.Sleep(conf.Delay)
		}
	}

	return nil, fmt.Errorf("could not connect to database after %d attempts: %w", conf.MaxAttempts, err)
}
