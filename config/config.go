package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Db    DbConfig
	TG    TgConfig
	Admin AdminsConfig
}

type DbConfig struct {
	Dsn string
}

type TgConfig struct {
	Token string
}

type AdminsConfig struct {
	AdminsID []int64
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

// LoadAminsID достаем IDшники админов из env
func LoadAminsID() []int64 {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Problem with load configs .env file. Using default config", err)
	}

	raw := os.Getenv("ADMINS_ID")
	if raw == "" {
		log.Println("ADMINS_ID is not set")
		return nil
	}

	admins := strings.Split(raw, ",")
	var res []int64

	for _, strID := range admins {
		strID = strings.TrimSpace(strID)
		if strID == "" {
			continue
		}

		id, err := strconv.Atoi(strID)
		if err != nil {
			log.Printf("invalid admin ID '%s': %v", strID, err)
			continue
		}

		res = append(res, int64(id))
	}

	return res

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
		Admin: AdminsConfig{
			AdminsID: LoadAminsID(),
		},
	}
}
