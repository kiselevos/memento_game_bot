package middleware

import (
	"log"

	"github.com/kiselevos/photo_battle_bot/internal/botinterface"

	"gopkg.in/telebot.v3"
)

func OnlyAdmins(bot botinterface.BotInterface) func(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			chat := c.Chat()
			user := c.Sender()

			// Пропускаем приватные чаты
			if chat.Type == telebot.ChatPrivate {
				return next(c)
			}

			// Проверка роли пользователя
			member, err := bot.ChatMemberOf(chat, user)
			if err != nil {
				log.Printf("[MIDDLEWARE] Ошибка ChatMemberOf: %v", err)
				// Можно тоже отправить алерт с ошибкой
				if c.Callback() != nil {
					return c.Respond(&telebot.CallbackResponse{
						Text: "⚠️ Не удалось проверить доступ.",
					})
				}
				return nil
			}

			if member.Role == telebot.Administrator || member.Role == telebot.Creator {
				return next(c)
			}

			// Если это callback, показываем алерт
			if c.Callback() != nil {
				return c.Respond(&telebot.CallbackResponse{
					Text: "🚫 Только администратор может использовать эту кнопку.",
				})
			}

			// Для обычных сообщений (на всякий случай)
			return c.Reply("🚫 Только администратор может использовать эту команду.")
		}
	}
}
