package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
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
}

func GetDsn() string {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Problem with load configs .env file. Using default config", err)
	}

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
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
		},
		Db: DbConfig{
			Dsn: GetDsn(),
		},
	}
}
