package middleware

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/botinterface"
	"errors"
	"log"

	"gopkg.in/telebot.v3"
)

// CheckBotAdminRights - проверка является ли бот админом
func CheckBotAdminRights(c telebot.Context, botUser *telebot.User, bot botinterface.BotInterface) error {

	chat := c.Chat()

	member, err := bot.ChatMemberOf(chat, botUser)
	if err != nil {
		log.Printf("[ERROR] Не удалось получить статус бота в чате: %v", err)
		c.Send(messages.ErrorMessagesForUser)
		return errors.New("не удалось получить статус бота")
	}

	if member.Role != telebot.Administrator {
		log.Printf("[WARN] Бот не является админом в чате %d (роль: %s)", chat.ID, member.Role)
		c.Send(messages.BotIsNotAdmin)
		return errors.New("Бот не админ")
	}

	return nil

}
