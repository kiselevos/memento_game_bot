package bot

import (
	"PhotoBattleBot/internal/game"

	"gopkg.in/telebot.v3"
)

func InitRouters(bot *telebot.Bot, gm *game.GameManager) {
	handlers := NewHandlers(bot, gm)
	handlers.Register()
}
