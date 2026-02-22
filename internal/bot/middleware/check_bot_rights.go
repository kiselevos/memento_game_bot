package middleware

import (
	"errors"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/botinterface"

	"gopkg.in/telebot.v3"
)

var (
	ErrBotNotAdmin          = errors.New("bot is not admin")
	ErrBotStatusUnavailable = errors.New("bot status unavailable")
)

// CheckBotAdminRights - проверка является ли бот админом
func CheckBotAdminRights(c telebot.Context, botUser *telebot.User, bot botinterface.BotInterface) error {

	chat := c.Chat()

	if chat.Type == telebot.ChatPrivate {
		return nil
	}

	member, err := bot.ChatMemberOf(chat, botUser)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBotStatusUnavailable, err)
	}

	if member.Role != telebot.Administrator {
		return ErrBotNotAdmin
	}

	return nil
}
