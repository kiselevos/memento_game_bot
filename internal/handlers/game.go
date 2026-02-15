package handlers

import (
	"fmt"
	"log"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

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

	gh.Bot.Handle("/start", gh.Start, middleware.PrivateOnly(gh.Bot))
	gh.Bot.Handle("/startgame", gh.StartGame)
	gh.Bot.Handle("/endgame", gh.HandleEndGame, middleware.OnlyHost(gh.GameManager))

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
	return c.Send(messages.WelcomeSingleMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
}

// StartGame - работает из любого места, начинает новую сессию, заканчивая старую
func (gh *GameHandlers) StartGame(c telebot.Context) error {

	log.Printf("Callback from: %s", c.Sender().Username)
	err := c.Respond()
	if err != nil {
		log.Printf("Respond error: %v", err)
	}

	if err := middleware.CheckBotAdminRights(c, gh.BotInfo, gh.Bot); err != nil {
		log.Println("[ERROR] Запуск игры без прав админа", err)
		return err
	}

	chatID := c.Chat().ID
	user := game.GetUserFromTelebot(c.Sender())

	if gh.GameManager.CheckFirstGame(chatID) {
		if gh.Bot != nil {
			gh.Bot.Send(&telebot.Chat{ID: chatID}, messages.WelcomeGroupMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			bot.WaitingAnimation(c, gh.Bot, 5)
		}
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.RoundHandlers.StartRoundBtn}}

	text := fmt.Sprintf(messages.GameStartedWithHost, game.DisplayNameHTML(&user))

	gh.GameManager.StartNewGameSession(chatID, user)

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}

// HandleEndGame - завершение игры, подсчет результатов сесссии
func (gh *GameHandlers) HandleEndGame(c telebot.Context) error {
	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.StartGameBtn}}

	session, exist := gh.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, []telebot.InlineButton{gh.FeedbackHandlers.FeedbackBtn})

	result := bot.RenderScore(bot.FinalScore, session.TotalScore())

	gh.GameManager.EndGame(chatID)

	return c.Send(result+"\n"+messages.FinishGameMassage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
