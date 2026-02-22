package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/kiselevos/memento_game_bot/internal/config"
)

type Db struct{ *sql.DB }

func NewDB(ctx context.Context, conf *config.DbConfig) (*Db, error) {
	var lastErr error

	for i := 1; i <= conf.MaxAttempts; i++ {
		db, err := sql.Open("pgx", conf.Dsn)
		if err != nil {
			lastErr = fmt.Errorf("open (attempt=%d): %w", i, err)
			if err := sleepOrDone(ctx, conf.Delay); err != nil {
				return nil, fmt.Errorf("db init canceled: %w (last: %v)", err, lastErr)
			}
			continue
		}

		db.SetMaxOpenConns(conf.MaxOpenConns)
		db.SetMaxIdleConns(conf.MaxIdleConns)
		db.SetConnMaxLifetime(conf.ConnMaxLifetime)

		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err == nil {
			return &Db{db}, nil
		}

		_ = db.Close() // закрываем нерабочее подключение

		lastErr = fmt.Errorf("ping (attempt=%d): %w", i, err)

		if err := sleepOrDone(ctx, conf.Delay); err != nil {
			return nil, fmt.Errorf("db init canceled: %w (last: %v)", err, lastErr)
		}
	}

	return nil, fmt.Errorf("could not connect to database after %d attempts: %w", conf.MaxAttempts, lastErr)
}

func sleepOrDone(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
