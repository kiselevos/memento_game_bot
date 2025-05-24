package middleware

import "gopkg.in/telebot.v3"

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
