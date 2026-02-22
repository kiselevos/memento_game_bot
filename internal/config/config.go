package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Db     DbConfig
	TG     TgConfig
	Admin  AdminsConfig
	Bot    BotConfig
	Logger LogConfig
}

type DbConfig struct {
	Dsn             string
	MaxAttempts     int
	Delay           time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type TgConfig struct {
	Token string
}

type AdminsConfig struct {
	AdminsID []int64
}

type BotConfig struct {
	DropOldMessagesAfter time.Duration
	FeedbackTTL          time.Duration
}

type LogConfig struct {
	Level  slog.Level
	AppEnv string
}

// Logging
func GetLogConfig() LogConfig {

	appEnv := os.Getenv("APP_ENV")

	if appEnv != "local" {
		appEnv = "prod"
	}

	return LogConfig{
		Level:  levelFromEnv(),
		AppEnv: appEnv,
	}
}

func levelFromEnv() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Bot
func GetBotConfig() BotConfig {
	return BotConfig{
		DropOldMessagesAfter: envDuration("BOT_DROP_OLD_TIMEOUT", 10*time.Second),
		FeedbackTTL:          envDuration("BOT_FEEDBACK_TTL", 10*time.Minute),
	}
}

func GetDbConfig() (DbConfig, error) {
	dsn, err := GetDsn()
	if err != nil {
		return DbConfig{}, err
	}

	return DbConfig{
		Dsn:             dsn,
		Delay:           envDuration("DB_DELAY_CONNECTION", 2*time.Second),
		MaxAttempts:     envInt("DB_MAX_ATTEMPTS", 5),
		MaxOpenConns:    envInt("DB_MAX_OPEN_CONN", 10),
		MaxIdleConns:    envInt("DB_MAX_IDLE_CONN", 5),
		ConnMaxLifetime: envDuration("DB_MAX_LIFETIME_CONN", 30*time.Minute),
	}, nil
}

func GetDsn() (string, error) {
	env := os.Getenv("APP_ENV")

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
		if env == "docker" {
			host = "postgres"
		}
	}

	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

	if user == "" || pass == "" || port == "" || name == "" {
		return "", fmt.Errorf("db env is not set полностью: DB_USER, DB_PASSWORD, DB_PORT, DB_NAME are required")
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&TimeZone=UTC",
		user, pass, host, port, name,
	)
	return dsn, nil
}

func GetAdminConfig() AdminsConfig {
	raw := os.Getenv("ADMINS_ID")
	if strings.TrimSpace(raw) == "" {
		return AdminsConfig{AdminsID: nil}
	}

	parts := strings.Split(raw, ",")
	adminsID := make([]int64, 0, len(parts))

	for _, strID := range parts {
		strID = strings.TrimSpace(strID)
		if strID == "" {
			continue
		}
		id, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			continue
		}
		adminsID = append(adminsID, id)
	}

	return AdminsConfig{AdminsID: adminsID}
}

func LoadConfig() (*Config, error) {
	if os.Getenv("APP_ENV") != "docker" {
		_ = godotenv.Load() // тихо; отсутствие .env - норм
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN not set")
	}

	dbCfg, err := GetDbConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		TG:     TgConfig{Token: token},
		Db:     dbCfg,
		Admin:  GetAdminConfig(),
		Bot:    GetBotConfig(),
		Logger: GetLogConfig(),
	}, nil
}

// helper for duration
func envDuration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
