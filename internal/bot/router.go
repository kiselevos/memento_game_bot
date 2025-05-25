package bot

import (
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/repositories"
	"PhotoBattleBot/internal/tasks"
)

// func InitRouters(bot BotInterface, gm *game.GameManager, tl *tasks.TasksList, db *db.Db) {
// 	handlers := NewHandlers(bot, gm, tl)
// 	handlers.Register()
// }

func InitRouters(
	bot BotInterface,
	gm *game.GameManager,
	tl *tasks.TasksList,
	userRepo *repositories.UserRepository,
	sessionRepo *repositories.SessionRepository,
	taskRepo *repositories.TaskRepository) {
	handlers := NewHandlers(bot, gm, tl, userRepo, sessionRepo, taskRepo)
	handlers.Register()
}
