package main

import (
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/logging"
	"PhotoBattleBot/internal/tasks"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	tb "gopkg.in/telebot.v3"
)

func main() {

	logging.InitLogger("bot.log")

	err := godotenv.Load()
	if err != nil {
		log.Println("Problem with .env file")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN not set")
	}

	pref := tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
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

	bot.InitRouters(b, gm, tl)

	log.Println("Bot starts...")
	b.Start()
}
