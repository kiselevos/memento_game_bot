package handlers

import (
	"errors"
	"fmt"
	"log/slog"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

	"gopkg.in/telebot.v3"
)

type RoundHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	GameHandlers *GameHandlers

	StartRoundBtn telebot.InlineButton
}

func NewRoundHandlers(bot botinterface.BotInterface, gm *game.GameManager) *RoundHandlers {

	h := &RoundHandlers{
		Bot:         bot,
		GameManager: gm,
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

	chatID := c.Chat().ID

	log := slog.Default().With(
		"chat_id", chatID,
		"user_id", c.Sender().ID,
		"action", "start_round",
		"is_callback", c.Callback() != nil,
	)

	if c.Callback() != nil {
		_ = c.Respond()

		empty := &telebot.ReplyMarkup{}
		if _, err := rh.Bot.EditReplyMarkup(c.Message(), empty); err != nil {
			log.Warn("cannot remove keyboard", "err", err)
		}
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{rh.GameHandlers.StartGameBtn}}

	round, task, err := rh.GameManager.StartNewRound(chatID, c.Sender().ID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrNoSession):
			return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)

		case errors.Is(err, game.ErrNoTasksLeft):
			_ = c.Send(messages.TheEndMessages, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			return rh.GameHandlers.HandleEndGame(c)

		case errors.Is(err, game.ErrWrongState):
			return c.Send(messages.RoundAlreadyStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML})

		case errors.Is(err, game.ErrOnlyHost):
			text := fmt.Sprintf(messages.OnlyHostRules)
			if c.Callback() != nil {
				_ = c.Respond(&telebot.CallbackResponse{
					Text: text,
				})
				return nil
			}
			return c.Reply(text)

		default:
			log.Error("failed to start new round", "err", err)
			_ = c.Send(messages.ErrorMessagesForUser, &telebot.SendOptions{ParseMode: telebot.ModeHTML})

			return err
		}
	}

	roundMsg := fmt.Sprintf(messages.RoundStartedMessage, round)
	text := roundMsg + "\n<b>" + task + "</b>"

	btn := rh.StartRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
