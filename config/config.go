package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
	env := os.Getenv("APP_ENV")

	host := "localhost"
	if env == "docker" {
		host = "postgres"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&TimeZone=UTC",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		host,
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	fmt.Println("DSN used:", dsn)

	return dsn
}

func LoadAminsID() []int64 {
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
		id, err := strconv.ParseInt(strID, 10, 64)
		if err != nil {
			log.Printf("invalid admin ID '%s': %v", strID, err)
			continue
		}
		res = append(res, id)
	}
	return res
}

func LoadConfig() *Config {
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
