package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"

	"gopkg.in/telebot.v3"
)

type ScoreHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	startRoundBtn telebot.InlineButton
}

func NewScoreHandlers(bot botinterface.BotInterface, gm *game.GameManager) *ScoreHandlers {

	sh := &ScoreHandlers{
		Bot:         bot,
		GameManager: gm,
	}
	sh.startRoundBtn = telebot.InlineButton{
		Unique: "start_round",
		Text:   "Начать раунд",
	}
	return sh
}

func (sh *ScoreHandlers) Register() {

	sh.Bot.Handle("/score", sh.HandleScore)

	// Для прод версии
	// h.Bot.Handle("/score", GroupOnly(h.HandleScore))

}

// HandleScore - показать общий счет данной сессии
func (sh *ScoreHandlers) HandleScore(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := sh.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted)
	}
	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{sh.startRoundBtn}}

	result := bot.RenderScore(bot.GameScore, session.TotalScore())
	return c.Send(result, markup)
}
