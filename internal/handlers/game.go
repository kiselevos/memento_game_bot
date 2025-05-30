package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/bot/middleware"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"log"

	"gopkg.in/telebot.v3"
)

type GameHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager
	BotInfo     *telebot.User

	FeedbackHandlers *FeedbackHandlers
	RoundHandlers    *RoundHandlers

	StartGameBtn telebot.InlineButton
}

func NewGameHandlers(bot botinterface.BotInterface, gm *game.GameManager, botInfo *telebot.User) *GameHandlers {

	h := &GameHandlers{
		Bot:         bot,
		GameManager: gm,
		BotInfo:     botInfo,
	}
	h.StartGameBtn = telebot.InlineButton{
		Unique: "start_game",
		Text:   "Новая игра",
	}
	return h
}

func (gh *GameHandlers) Register() {

	gh.Bot.Handle("/start", gh.Start)
	gh.Bot.Handle("/startgame", gh.StartGame)
	gh.Bot.Handle("/endgame", gh.HandleEndGame)

	gh.Bot.Handle(&gh.StartGameBtn, gh.StartGame)

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
	return c.Send(messages.WelcomeSingleMessage, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

// StartGame - работает из любого места, начинает новую сессию, заканчивая старую
func (gh *GameHandlers) StartGame(c telebot.Context) error {

	if err := middleware.CheckBotAdminRights(c, gh.BotInfo, gh.Bot); err != nil {
		log.Println("[ERROR] Запуск игры без прав админа", err)
		return err
	}

	chatID := c.Chat().ID

	if gh.GameManager.CheckFirstGame(chatID) {
		if gh.Bot != nil {
			gh.Bot.Send(&telebot.Chat{ID: chatID}, messages.WelcomeGroupMessage, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
			bot.WaitingAnimation(c, gh.Bot, 5)
		}
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.RoundHandlers.StartRoundBtn}}

	gh.GameManager.StartNewGameSession(chatID)

	return c.Send(messages.GameRulesText, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
}

// HandleEndGame - завершение игры, подсчет результатов сесссии
func (gh *GameHandlers) HandleEndGame(c telebot.Context) error {
	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.StartGameBtn}}

	session, exist := gh.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, []telebot.InlineButton{gh.FeedbackHandlers.FeedbackBtn})

	result := bot.RenderScore(bot.FinalScore, session.TotalScore())

	gh.GameManager.EndGame(chatID)

	return c.Send(result+"\n"+messages.FinishGameMassage, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
}
