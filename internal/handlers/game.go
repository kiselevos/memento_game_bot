package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"

	"gopkg.in/telebot.v3"
)

type GameHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	FeedbackHandlers *FeedbackHandlers

	startRoundBtn telebot.InlineButton
}

func NewGameHandlers(bot botinterface.BotInterface, gm *game.GameManager) *GameHandlers {

	h := &GameHandlers{
		Bot:         bot,
		GameManager: gm,
	}
	h.startRoundBtn = telebot.InlineButton{
		Unique: "start_round",
		Text:   "Начать раунд",
	}
	return h
}

func (h *GameHandlers) Register() {

	h.Bot.Handle("/start", h.Start)
	h.Bot.Handle("/startgame", h.StartGame)
	h.Bot.Handle("/endgame", h.HandleEndGame)

	// Для прод версии
	// h.Bot.Handle("/startgame", GroupOnly(h.StartGame))
	// h.Bot.Handle("/endgame", GroupOnly(h.HandleEndGame))
}

// Start - приветствие, или приветствие для фидбэка если переход по кнопке.
func (gh *GameHandlers) Start(c telebot.Context) error {
	args := c.Args()

	if len(args) > 0 && args[0] == "feedback" {
		return gh.FeedbackHandlers.SendFeedbackInstructions(c)
	}
	return c.Send(messages.WelcomeSingleMessage)
}

// StartGame - работает из любого места, начинает новую сессию, заканчивая старую
func (gh *GameHandlers) StartGame(c telebot.Context) error {

	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.startRoundBtn}}

	gh.GameManager.StartNewGameSession(chatID)

	if gh.Bot != nil {
		gh.Bot.Send(&telebot.Chat{ID: chatID}, messages.WelcomeGroupMessage)
	}

	return c.Send(messages.GameRulesText, markup)
}

// HandleEndGame - завершение игры, подсчет результатов сесссии
func (h *GameHandlers) HandleEndGame(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted)
	}

	result := bot.RenderScore(bot.FinalScore, session.TotalScore())

	h.GameManager.EndGame(chatID)

	return c.Send(result + "\n\n" + messages.FinishGameMassage)
}
