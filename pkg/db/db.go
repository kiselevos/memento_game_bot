package db

import (
	"PhotoBattleBot/config"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Db struct {
	*gorm.DB
}

// NewDB - создание нового подключения к DB
func NewDB(conf *config.DbConfig) *Db {
	db, err := gorm.Open(postgres.Open(conf.Dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Problem with DB", err)
	}
	return &Db{db}
}
