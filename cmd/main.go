package main

import (
	"PhotoBattleBot/config"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/logging"
	"PhotoBattleBot/internal/tasks"
	"log"

	tb "gopkg.in/telebot.v3"
)

func main() {

	logging.InitLogger("bot.log")

	conf := config.LoadConfig()

	b, err := tb.NewBot(conf.TG.Pref)
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
