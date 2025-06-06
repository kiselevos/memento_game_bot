package middleware

import (
	"PhotoBattleBot/internal/botinterface"

	"gopkg.in/telebot.v3"
)

// GroupOnly - Обертка для групповых команд.
func GroupOnly(handler telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		chatType := c.Chat().Type
		if chatType != telebot.ChatGroup && chatType != telebot.ChatSuperGroup {
			return c.Send("Играть в одиночестве - не интересно. Добавь меня в группу друзей.")
		}
		return handler(c)
	}
}

func PrivateOnly(bot botinterface.BotInterface) func(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			chat := c.Chat()

			// Пропускаем приватные чаты
			if chat.Type == telebot.ChatPrivate {
				return next(c)
			}
			return nil
		}
	}
}
