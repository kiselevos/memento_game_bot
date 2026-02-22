package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf, err := config.LoadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log := logging.InitSlogLogger(conf.Logger)

	database, err := db.NewDB(ctx, &conf.Db)
	if err != nil {
		log.Error("startup failed", "err", err)
		os.Exit(1)
	}

	log.Info("database connected")

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
			l := log.With("action", "telebot_error")

			if c != nil {
				if chat := c.Chat(); chat != nil {
					l = l.With("chat_id", chat.ID)
				}
				if s := c.Sender(); s != nil {
					l = l.With("user_id", s.ID)
				}
				if m := c.Message(); m != nil {
					l = l.With("msg_id", m.ID)
				}
			}

			l.Error("telebot handler failed", "err", err)

			logging.Notify(slog.LevelError, "telebot handler failed", "err", err)
		},
	}

	b, err := tb.NewBot(pref)
	if err != nil {
		log.Error("startup failed: create bot", "err", err)
		os.Exit(1)
	}

	n := logging.NewNotifier(b, conf.Admin.AdminsID)
	logging.SetNotifier(n)

	botInfo := b.Me

	gm := game.NewGameManager(ctx, rec, tasks)
	fm := feedback.NewFeedbackManager(conf.Bot.FeedbackTTL)

	h := handlers.NewHandlers(b, fm, feedbackRepo, conf.Admin.AdminsID, botInfo, gm)
	h.RegisterAll()

	log.Info("application starting")

	go func() {
		b.Start()
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	b.Stop()
	log.Info("bot stopped")

	dbClosed := make(chan struct{})
	go func() {
		defer close(dbClosed)
		if err := database.Close(); err != nil {
			log.Warn("database close failed", "err", err)
		} else {
			log.Info("database closed")
		}
	}()

	select {
	case <-dbClosed:
	case <-shutdownCtx.Done():
		log.Warn("shutdown timeout reached; forcing exit")
		os.Exit(1)
	}

	log.Info("shutdown completed")
}
