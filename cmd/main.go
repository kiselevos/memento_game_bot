package main

import (
	"PhotoBattleBot/config"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/bot/middleware"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/logging"
	"PhotoBattleBot/internal/repositories"
	"PhotoBattleBot/internal/tasks"
	"PhotoBattleBot/pkg/db"
	"log"
	"time"

	tb "gopkg.in/telebot.v3"
)

func main() {

	logging.InitLogger("bot.log")

	conf := config.LoadConfig()

	database := db.NewDB(conf)

	// Repository
	userRepo := repositories.NewUserRepository(database)
	sessionRepo := repositories.NewSessionRepository(database)
	taskRepo := repositories.NewTaskRepository(database)

	// Tg settings
	pref := tb.Settings{
		Token:  conf.TG.Token,
		Poller: middleware.DropOldMessages(10 * time.Second),
		OnError: func(err error, c tb.Context) {
			log.Printf("Error: %v\n", err)
		},
	}

	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация GameManager
	tl, err := tasks.NewTasksList("assets/tasks.json")
	if err != nil {
		log.Fatal(err)
	}
	gm := game.NewGameManager()

	bot.InitRouters(b, gm, tl, userRepo, sessionRepo, taskRepo)

	log.Println("Bot starts...")
	b.Start()
}
