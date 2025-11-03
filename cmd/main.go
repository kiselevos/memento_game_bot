package main

import (
	"log"
	"time"

	"github.com/kiselevos/memento_game_bot/config"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/feedback"
	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/handlers"
	"github.com/kiselevos/memento_game_bot/internal/logging"
	"github.com/kiselevos/memento_game_bot/internal/repositories"
	"github.com/kiselevos/memento_game_bot/internal/tasks"
	"github.com/kiselevos/memento_game_bot/pkg/db"

	tb "gopkg.in/telebot.v3"
)

func main() {

	logging.InitLogger("logs/bot.log")

	conf := config.LoadConfig()

	database, err := db.NewDB(conf)
	if err != nil {
		log.Fatal(err)
	}

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

	botInfo := b.Me

	// Инициализация GameManager
	tl, err := tasks.NewTasksList("assets/tasks.json")
	if err != nil {
		log.Fatal(err)
	}
	gm := game.NewGameManager(userRepo, sessionRepo, taskRepo)
	fm := feedback.NewFeedbackManager(10 * time.Minute)

	h := handlers.NewHandlers(b, fm, conf.Admin.AdminsID, botInfo, gm, tl)
	h.RegisterAll()

	log.Println("Bot starts...")
	b.Start()
}
