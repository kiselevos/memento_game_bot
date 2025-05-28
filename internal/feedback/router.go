package feedback

import (
	"PhotoBattleBot/internal/botinterface"
)

func InitRouters(bot botinterface.BotInterface, fm *FeedbackManager, adminsID []int64, botUN string) {
	handlers := NewFeedbackHandler(bot, fm, adminsID, botUN)
	handlers.Register()
}
