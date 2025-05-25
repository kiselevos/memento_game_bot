package config

import (
	"PhotoBattleBot/internal/bot/middleware"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/telebot.v3"
)

type Config struct {
	Db DbConfig
	TG TgConfig
}

type DbConfig struct {
	Dsn string
}

type TgConfig struct {
	Token string
	Pref  telebot.Settings
}

func getDsn() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Problem with load configs .env file. Using default config", err)
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN not set")
	}

	return &Config{
		TG: TgConfig{
			Token: token,
			Pref: telebot.Settings{
				Token:  token,
				Poller: middleware.DropOldMessages(10 * time.Second),
				OnError: func(err error, c telebot.Context) {
					log.Printf("Error: %v\n", err)
				},
			},
		},
		Db: DbConfig{
			Dsn: getDsn(),
		},
	}
}
