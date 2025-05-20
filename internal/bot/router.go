package bot

import (
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"

	"gopkg.in/telebot.v3"
)

func InitRouters(bot *telebot.Bot, gm *game.GameManager, tl *tasks.TasksList) {
	handlers := NewHandlers(bot, gm, tl)
	handlers.Register()
}
