package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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

	if c.Callback() != nil {
		_ = c.Respond()

		empty := &telebot.ReplyMarkup{}
		if _, err := rh.Bot.EditReplyMarkup(c.Message(), empty); err != nil {
			log.Printf("[WARN] cannot remove keyboard: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{rh.GameHandlers.StartGameBtn}}

	chatID := c.Chat().ID

	round, task, err := rh.GameManager.StartNewRound(ctx, chatID)
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

	roundMsg := fmt.Sprintf(messages.RoundStartedMessage, round)
	text := roundMsg + "\n<b>" + task + "</b>"

	btn := rh.StartRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
