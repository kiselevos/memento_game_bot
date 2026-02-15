package middleware

import (
	"fmt"
	"log"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/game"

	"gopkg.in/telebot.v3"
)

// Проверка на админа заменена хостом
// func OnlyAdmins(bot botinterface.BotInterface) func(next telebot.HandlerFunc) telebot.HandlerFunc {
// 	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
// 		return func(c telebot.Context) error {
// 			chat := c.Chat()
// 			user := c.Sender()

// 			// Пропускаем приватные чаты
// 			if chat.Type == telebot.ChatPrivate {
// 				return next(c)
// 			}

// 			// Проверка роли пользователя
// 			member, err := bot.ChatMemberOf(chat, user)
// 			if err != nil {
// 				log.Printf("[MIDDLEWARE] Ошибка ChatMemberOf: %v", err)
// 				// алерт с ошибкой
// 				if c.Callback() != nil {
// 					return c.Respond(&telebot.CallbackResponse{
// 						Text: "⚠️ Не удалось проверить доступ.",
// 					})
// 				}
// 				return nil
// 			}

// 			if member.Role == telebot.Administrator || member.Role == telebot.Creator {
// 				return next(c)
// 			}

// 			// Если это callback, показываем алерт
// 			if c.Callback() != nil {
// 				return c.Respond(&telebot.CallbackResponse{
// 					Text: "🚫 Только администратор может использовать эту кнопку.",
// 				})
// 			}

// 			// Для обычных сообщений (на всякий случай)
// 			return c.Reply("🚫 Только администратор может использовать эту команду.")
// 		}
// 	}
// }

func OnlyHost(gm *game.GameManager) func(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) error {
			chat := c.Chat()
			user := c.Sender()

			// Пропускаем приватные чаты
			if chat.Type == telebot.ChatPrivate {
				return next(c)
			}

			// Достаем сессию
			session, exist := gm.GetSession(chat.ID)
			if !exist {
				log.Printf("[INFO] Попытка запуска раунда без начала новой игры в чате %d", chat.ID)
				if c.Callback() != nil {
					_ = c.Respond()
					return c.Respond(&telebot.CallbackResponse{
						Text: messages.GameNotStarted,
					})
				}

				return c.Reply(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			}

			if session.IsHost(user.ID) {
				return next(c)
			}

			text := fmt.Sprintf(messages.OnlyHostRules, session.Host.FirstName)

			if c.Callback() != nil {
				return c.Respond(&telebot.CallbackResponse{
					Text: text,
				})
			}

			// На всякий случай, хотя в основном inline
			return c.Reply(text)
		}
	}
}
