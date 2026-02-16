package handlers

import (
	"errors"
	"log"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/tasks"

	"gopkg.in/telebot.v3"
)

type RoundHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager
	TasksList   *tasks.TasksList

	GameHandlers *GameHandlers

	StartRoundBtn telebot.InlineButton
}

func NewRoundHandlers(bot botinterface.BotInterface, gm *game.GameManager, tl *tasks.TasksList) *RoundHandlers {

	h := &RoundHandlers{
		Bot:         bot,
		GameManager: gm,
		TasksList:   tl,
	}
	h.StartRoundBtn = telebot.InlineButton{
		Unique: "start_round",
		Text:   "Начать раунд",
	}
	return h
}

func (rh *RoundHandlers) Register() {

	rh.Bot.Handle(&rh.StartRoundBtn, rh.HandleStartRound, middleware.OnlyHost(rh.GameManager))
	rh.Bot.Handle("/newround", rh.HandleStartRound, middleware.OnlyHost(rh.GameManager))
}

func (rh *RoundHandlers) HandleStartRound(c telebot.Context) error {

	log.Printf("Callback from: %s", c.Sender().Username)

	if c.Callback() != nil {
		_ = c.Respond()
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{rh.GameHandlers.StartGameBtn}}

	chatID := c.Chat().ID

	task, err := rh.GameManager.StartNewRound(chatID, rh.TasksList)
	if err != nil {
		if errors.Is(err, game.ErrNoSession) {
			return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
		}
		if errors.Is(err, game.ErrNoTasksLeft) {
			_ = c.Send(messages.TheEndMessages, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			return rh.GameHandlers.HandleEndGame(c)
		}

		log.Printf("[ERROR] Ошибка начала нового раунда %d: %v", chatID, err)
		return c.Send(messages.ErrorMessagesForUser, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
	}

	text := messages.RoundStartedMessage + "\n<b>" + task + "</b>"

	btn := rh.StartRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
