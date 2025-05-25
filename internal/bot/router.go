package bot

import (
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"PhotoBattleBot/pkg/db"
)

func InitRouters(bot BotInterface, gm *game.GameManager, tl *tasks.TasksList, db *db.Db) {
	handlers := NewHandlers(bot, gm, tl)
	handlers.Register()
}
