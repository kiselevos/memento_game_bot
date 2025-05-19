package main

import (
	"PhotoBattleBot/internal/bot"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/telebot.v3"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Problem with .env file")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN not set")
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c telebot.Context) {
			log.Printf("Error: %v\n", err)
		},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	bot.RegisterHandlers(b)

	log.Println("Bot starts...")
	b.Start()
}
