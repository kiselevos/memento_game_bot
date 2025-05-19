package bot

import (
	"PhotoBattleBot/internal/messages"

	"gopkg.in/telebot.v3"
)

func handleStart(c telebot.Context) error {
	return c.Send(messages.WelcomeMessage)
}

func handleResetGame(c telebot.Context) error {
	return c.Send((messages.GameResetMessage))
}

func handleHelp(c telebot.Context) error {
	return c.Send(messages.HelpMessage)
}

func handleStartRound(c telebot.Context) error {
	return c.Send(messages.RoundStartedMessage)
}
