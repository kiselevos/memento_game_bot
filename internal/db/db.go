package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kiselevos/memento_game_bot/internal/config"
)

type Db struct {
	*sql.DB
}

// NewDB - создание нового подключения к DB
func NewDB(conf *config.DbConfig) (*Db, error) {

	var db *sql.DB
	var err error

	for i := 1; i <= conf.MaxAttempts; i++ {

		db, err = sql.Open("pgx", conf.Dsn)
		if err != nil {
			log.Println("Failed to open DB:", err)
			time.Sleep(conf.Delay)
			continue
		}

		db.SetMaxOpenConns(conf.MaxOpenConns)
		db.SetMaxIdleConns(conf.MaxIdleConns)
		db.SetConnMaxLifetime(conf.ConnMaxLifetime)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			log.Println("Database connected successfully")
			return &Db{db}, nil
		}

		log.Println("Database ping failed, retrying...",
			"attempt", i, "error", err)

		time.Sleep(conf.Delay)
	}

	return nil, fmt.Errorf(
		"could not connect to database after %d attempts: %w",
		conf.MaxAttempts, err,
	)

}
