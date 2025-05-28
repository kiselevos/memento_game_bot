package bot

import (
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
)

func InitRouters(bot botinterface.BotInterface, gm *game.GameManager, tl *tasks.TasksList) {
	handlers := NewHandlers(bot, gm, tl)
	handlers.Register()
}
