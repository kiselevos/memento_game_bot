package bot

import (
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
)

func InitRouters(bot BotInterface, gm *game.GameManager, tl *tasks.TasksList) {
	handlers := NewHandlers(bot, gm, tl)
	handlers.Register()
}
