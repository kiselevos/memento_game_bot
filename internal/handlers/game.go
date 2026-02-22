package handlers

import (
	"errors"
	"fmt"
	"log/slog"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/logging"

	"gopkg.in/telebot.v3"
)

type GameHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager
	BotInfo     *telebot.User

	FeedbackHandlers *FeedbackHandlers
	RoundHandlers    *RoundHandlers

	// inline buttons
	StartGameBtn      telebot.InlineButton
	ConfirmRestartBtn telebot.InlineButton
	CancelRestartBtn  telebot.InlineButton
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
	h.ConfirmRestartBtn = telebot.InlineButton{
		Unique: "confirm_new_game",
		Text:   "🆕 Начать новую игру",
	}
	h.CancelRestartBtn = telebot.InlineButton{
		Unique: "cancel_new_game",
		Text:   "❌ Отмена",
	}
	return h
}

func (gh *GameHandlers) Register() {

	gh.Bot.Handle("/start", gh.Start, middleware.PrivateOnly(gh.Bot))
	gh.Bot.Handle("/startgame", gh.StartGame)
	gh.Bot.Handle("/endgame", gh.HandleEndGame, middleware.OnlyHost(gh.GameManager))

	gh.Bot.Handle(&gh.StartGameBtn, gh.StartGame)
	gh.Bot.Handle(&gh.ConfirmRestartBtn, gh.ConfirmNewGame)
	gh.Bot.Handle(&gh.CancelRestartBtn, gh.CancelRestart)

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

	if c.Callback() != nil {
		_ = c.Respond()
	}

	if err := middleware.CheckBotAdminRights(c, gh.BotInfo, gh.Bot); err != nil {
		switch {
		case errors.Is(err, middleware.ErrBotNotAdmin):
			_ = c.Send(messages.BotIsNotAdmin)
			return nil

		case errors.Is(err, middleware.ErrBotStatusUnavailable):
			_ = c.Send(messages.ErrorMessagesForUser)
			return err

		default:
			_ = c.Send(messages.ErrorMessagesForUser)
			return err
		}
	}

	chatID := c.Chat().ID
	user := game.GetUserFromTelebot(c.Sender())

	var hostName string
	err := gh.GameManager.DoWithSession(chatID, func(s *game.GameSession) error {
		hostName = s.Host.FirstName
		return nil
	})

	if err != nil {
		if gh.GameManager.CheckFirstGame(chatID) {
			if gh.Bot != nil {
				gh.Bot.Send(&telebot.Chat{ID: chatID}, messages.WelcomeGroupMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
				bot.WaitingAnimation(c, gh.Bot, 5)
			}
		}

		markup := &telebot.ReplyMarkup{}
		markup.InlineKeyboard = [][]telebot.InlineButton{{gh.RoundHandlers.StartRoundBtn}}

		text := fmt.Sprintf(messages.GameStartedWithHost, game.DisplayNameHTML(&user))

		fallbackUsed, err := gh.GameManager.StartNewGameSession(chatID, user)
		if err != nil {
			_ = c.Send(messages.ErrorMessagesForUser, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			return err
		}

		if fallbackUsed {
			gh.notifyDBFallback(chatID, user.ID, "start_game")
		}

		return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{
		{gh.ConfirmRestartBtn, gh.CancelRestartBtn},
	}

	text := fmt.Sprintf(messages.GameAlreadyExist, hostName)

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}

// Подтверждение при рестарте.
func (gh *GameHandlers) ConfirmNewGame(c telebot.Context) error {
	if c.Callback() != nil {
		_ = c.Respond(&telebot.CallbackResponse{Text: messages.RestartGameMsg})
	}

	chatID := c.Chat().ID
	user := game.GetUserFromTelebot(c.Sender())

	fallbackUsed, err := gh.GameManager.StartNewGameSession(chatID, user)
	if err != nil {
		_ = c.Send(messages.ErrorMessagesForUser, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
		return err // OnError -> Notify
	}

	if fallbackUsed {
		gh.notifyDBFallback(chatID, user.ID, "start_game")
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.RoundHandlers.StartRoundBtn}}

	text := fmt.Sprintf(messages.GameStartedWithHost, game.DisplayNameHTML(&user))
	return c.Edit(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}

// Отмена рестарта
func (gh *GameHandlers) CancelRestart(c telebot.Context) error {
	if c.Callback() != nil {
		_ = c.Respond(&telebot.CallbackResponse{Text: messages.CancelRestartMsg})
		_ = c.Edit(messages.ContinueGameMsg)
		return nil
	}
	return nil
}

// HandleEndGame - завершение игры, подсчет результатов сесссии
func (gh *GameHandlers) HandleEndGame(c telebot.Context) error {
	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{gh.StartGameBtn}}

	totalScore, err := gh.GameManager.GetTotalScore(chatID)

	if err != nil {
		return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, []telebot.InlineButton{gh.FeedbackHandlers.FeedbackBtn})

	result := bot.RenderScore(bot.FinalScore, totalScore)

	err = gh.GameManager.EndGame(chatID, c.Sender().ID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrNoSession):
			return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)

		default:
			_ = c.Send(result+"\n"+messages.FinishGameMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
			return err
		}
	}

	return c.Send(result+"\n"+messages.FinishGameMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}

// Helper для логировани
func (gh *GameHandlers) notifyDBFallback(chatID int64, userID int64, action string) {
	slog.Default().Error("database unavailable, fallback tasks used",
		"chat_id", chatID,
		"user_id", userID,
		"action", action,
	)

	logging.Notify(slog.LevelError, "database unavailable, fallback tasks used",
		"chat_id", chatID,
		"user_id", userID,
		"action", action,
	)
}
