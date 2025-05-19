package bot

import "gopkg.in/telebot.v3"

func RegisterHandlers(b *telebot.Bot) {
	b.Handle("/start", handleStart)
	b.Handle("/startgame", handleStartRound)
	b.Handle("/resetgame", handleResetGame)
	b.Handle("/help", handleHelp)
}
