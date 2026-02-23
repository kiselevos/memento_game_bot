package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kiselevos/memento_game_bot/internal/db"
)

var (
	testDatabase *db.Db
	sqlDB        *sql.DB
	pgC          *postgres.PostgresContainer
)

func newRecorderForTest() *Recorder {
	userRepo := NewUserRepo(testDatabase)
	sessionRepo := NewSessionRepo(testDatabase)
	taskRepo := NewTaskRepo(testDatabase)
	return NewRecorder(userRepo, taskRepo, sessionRepo)
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	pgC, err = postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic(fmt.Errorf("start postgres container: %w", err))
	}

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = dumpContainerLogs(ctx, pgC)
		panic(fmt.Errorf("connection string: %w", err))
	}

	sqlDB, err = sql.Open("pgx", dsn)
	if err != nil {
		_ = dumpContainerLogs(ctx, pgC)
		panic(fmt.Errorf("sql open: %w", err))
	}

	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetConnMaxLifetime(time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := pingWithRetry(pingCtx, sqlDB, 250*time.Millisecond); err != nil {
		_ = dumpContainerLogs(ctx, pgC)
		panic(fmt.Errorf("db ping: %w", err))
	}

	// ✅ создаём твой тип db.Db, который ожидают репозитории
	testDatabase = &db.Db{
		DB: sqlDB,
	}

	// Goose migrations
	if err := goose.SetDialect("postgres"); err != nil {
		_ = dumpContainerLogs(ctx, pgC)
		panic(fmt.Errorf("goose dialect: %w", err))
	}

	// ✅ поправь путь под свой проект
	migrationsDir := "../../migrations"

	if err := goose.Up(testDatabase.DB, migrationsDir); err != nil {
		_ = dumpContainerLogs(ctx, pgC)
		panic(fmt.Errorf("goose up: %w", err))
	}

	code := m.Run()

	_ = sqlDB.Close()
	_ = pgC.Terminate(ctx)

	os.Exit(code)
}

func pingWithRetry(ctx context.Context, dbConn *sql.DB, step time.Duration) error {
	var lastErr error
	for {
		if err := dbConn.PingContext(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping timeout: %w (last error: %v)", ctx.Err(), lastErr)
		case <-time.After(step):
			if step < 2*time.Second {
				step *= 2
			}
		}
	}
}

func dumpContainerLogs(ctx context.Context, c *postgres.PostgresContainer) error {
	r, err := c.Logs(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	b, _ := io.ReadAll(r)
	fmt.Printf("\n--- postgres container logs ---\n%s\n--- end logs ---\n", string(b))
	return nil
}

func cleanDB(t *testing.T) {
	t.Helper()

	_, err := testDatabase.Exec(`
		TRUNCATE TABLE
			feedback,
			sessions,
			users,
			tasks
		RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
