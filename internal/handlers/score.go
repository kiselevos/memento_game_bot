package handlers

import (
	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

	"gopkg.in/telebot.v3"
)

type ScoreHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	RoundHandlers *RoundHandlers
	GameHandlers  *GameHandlers
}

func NewScoreHandlers(bot botinterface.BotInterface, gm *game.GameManager) *ScoreHandlers {

	sh := &ScoreHandlers{
		Bot:         bot,
		GameManager: gm,
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

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{sh.GameHandlers.StartGameBtn}}

	totalScore, err := sh.GameManager.GetTotalScore(chatID)

	if err != nil {
		return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}

	markup.InlineKeyboard = [][]telebot.InlineButton{{sh.RoundHandlers.StartRoundBtn}}

	result := bot.RenderScore(bot.GameScore, totalScore)
	return c.Send(result, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
