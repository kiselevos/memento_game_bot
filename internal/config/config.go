package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Db    DbConfig
	TG    TgConfig
	Admin AdminsConfig
	Bot   BotConfig
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

func GetBotConfig() BotConfig {
	return BotConfig{
		DropOldMessagesAfter: envDuration("BOT_DROP_OLD_TIMEOUT", 10*time.Second),
		FeedbackTTL:          envDuration("BOT_FEEDBACK_TTL", 10*time.Minute),
	}
}

func GetDbConfig() DbConfig {
	return DbConfig{
		Dsn:             GetDsn(),
		Delay:           envDuration("DB_DELAY_CONNECTION", 2*time.Second),
		MaxAttempts:     envInt("DB_MAX_ATTEMPTS", 5),
		MaxOpenConns:    envInt("DB_MAX_OPEN_CONN", 10),
		MaxIdleConns:    envInt("DB_MAX_IDLE_CONN", 5),
		ConnMaxLifetime: envDuration("DB_MAX_LIFETIME_CONN", 30*time.Minute),
	}
}

func GetDsn() string {
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
		log.Fatal("DB env is not set полностью: DB_USER, DB_PASSWORD, DB_PORT, DB_NAME are required")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&TimeZone=UTC",
		user, pass, host, port, name,
	)

	return dsn
}

func GetAdminConfig() AdminsConfig {

	raw := os.Getenv("ADMINS_ID")
	if raw == "" {
		log.Println("ADMINS_ID is not set")
		return AdminsConfig{
			AdminsID: nil,
		}
	}

	admins := strings.Split(raw, ",")
	var adminsID []int64

	for _, strID := range admins {
		strID = strings.TrimSpace(strID)
		if strID == "" {
			continue
		}
		id, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			log.Printf("invalid admin ID '%s': %v", strID, err)
			continue
		}
		adminsID = append(adminsID, id)
	}

	return AdminsConfig{
		AdminsID: adminsID,
	}
}

func LoadConfig() (*Config, error) {

	if os.Getenv("APP_ENV") != "docker" {
		if err := godotenv.Load(); err != nil {
			log.Println(".env file not found, using system environment variables")
		}
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN not set")
	}

	return &Config{
		TG: TgConfig{
			Token: token,
		},
		Db:    GetDbConfig(),
		Admin: GetAdminConfig(),
		Bot:   GetBotConfig(),
	}, nil
}

// helper for duration
func envDuration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid %s=%q: %v (using default %s)", key, raw, err, def)
		return def
	}
	if d <= 0 {
		log.Printf("invalid %s=%q: must be > 0 (using default %s)", key, raw, def)
		return def
	}
	return d
}

// helper for int
func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid %s=%q: %v (using default %d)", key, raw, err, def)
		return def
	}
	if n <= 0 {
		log.Printf("invalid %s=%q: must be > 0 (using default %d)", key, raw, def)
		return def
	}
	return n
}
