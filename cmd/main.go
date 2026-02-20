package main

import (
	"log"

	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/config"
	"github.com/kiselevos/memento_game_bot/internal/db"
	"github.com/kiselevos/memento_game_bot/internal/feedback"
	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/handlers"
	"github.com/kiselevos/memento_game_bot/internal/logging"
	"github.com/kiselevos/memento_game_bot/internal/repositories"

	tb "gopkg.in/telebot.v3"
)

func main() {

	logging.InitLogger("logs/bot.log")

	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	database, err := db.NewDB(&conf.Db)
	if err != nil {
		log.Fatal(err)
	}

	// Repository
	userRepo := repositories.NewUserRepo(database)
	sessionRepo := repositories.NewSessionRepo(database)
	taskRepo := repositories.NewTaskRepo(database)
	feedbackRepo := repositories.NewFeedbackRepo(database)

	rec := repositories.NewRecorder(userRepo, taskRepo, sessionRepo)
	tasks := repositories.NewTaskStore(taskRepo)

	// Tg settings
	pref := tb.Settings{
		Token:  conf.TG.Token,
		Poller: middleware.DropOldMessages(conf.Bot.DropOldMessagesAfter),
		OnError: func(err error, c tb.Context) {
			log.Printf("Error: %v\n", err)
		},
	}

	b, err := tb.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	botInfo := b.Me

	gm := game.NewGameManager(rec, tasks)
	fm := feedback.NewFeedbackManager(conf.Bot.FeedbackTTL)

	h := handlers.NewHandlers(b, fm, feedbackRepo, conf.Admin.AdminsID, botInfo, gm)
	h.RegisterAll()

	log.Println("Bot starts...")
	b.Start()
}
