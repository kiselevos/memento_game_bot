package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/feedback"
	"github.com/kiselevos/memento_game_bot/internal/repositories"

	"gopkg.in/telebot.v3"
)

type FeedbackHandlers struct {
	Bot             botinterface.BotInterface
	FeedbackManager *feedback.FeedbackManager
	AdminsID        []int64
	BotUsername     string
	feedbackRepo    *repositories.FeedbackRepo

	FeedbackBtn telebot.InlineButton
}

func NewFeedbackHandler(
	bot botinterface.BotInterface,
	fm *feedback.FeedbackManager,
	adminsID []int64,
	botName string,
	fr *repositories.FeedbackRepo,
) *FeedbackHandlers {
	h := &FeedbackHandlers{
		Bot:             bot,
		FeedbackManager: fm,
		AdminsID:        adminsID,
		BotUsername:     botName,
		feedbackRepo:    fr,
	}

	h.FeedbackBtn = telebot.InlineButton{
		Text: "Оставить отзыв",
		URL:  fmt.Sprintf("https://t.me/%s?start=feedback", h.BotUsername),
	}

	return h
}

func (fh *FeedbackHandlers) Register() {
	fh.Bot.Handle("/feedback", fh.HandleStartFeedback)

	fh.Bot.Handle(telebot.OnText, fh.HandelFeedbackText)

	cancelBtn := &telebot.InlineButton{Unique: "cancel_feedback"}
	fh.Bot.Handle(cancelBtn, fh.HandelCancelFeedback)
}

func (fh *FeedbackHandlers) HandleStartFeedback(c telebot.Context) error {

	if c.Chat().Type == telebot.ChatPrivate {
		return fh.SendFeedbackInstructions(c)
	}

	inline := &telebot.ReplyMarkup{}
	inline.InlineKeyboard = [][]telebot.InlineButton{{fh.FeedbackBtn}}

	return c.Send(messages.StartFeedbackMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, inline)
}

func (fh *FeedbackHandlers) SendFeedbackInstructions(c telebot.Context) error {

	userID := c.Sender().ID

	fh.FeedbackManager.StartFeedback(userID)

	cancelBtn := telebot.InlineButton{Text: "Отменить отзыв", Unique: "cancel_feedback"}
	inline := &telebot.ReplyMarkup{}
	inline.InlineKeyboard = [][]telebot.InlineButton{{cancelBtn}}

	return c.Send(messages.AboutFeedback, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, inline)
}

func (fh *FeedbackHandlers) HandelCancelFeedback(c telebot.Context) error {
	userID := c.Sender().ID

	fh.FeedbackManager.CancelFeedback(userID)

	if err := c.Respond(&telebot.CallbackResponse{
		Text: "Отзыв отменён.",
	}); err != nil {
		log.Println("[ERROR] Не удалось отправить callback response:", err)
	}

	return c.Edit("Отправка отзыва отменена.")
}

func (fh *FeedbackHandlers) HandelFeedbackText(c telebot.Context) error {

	userID := c.Sender().ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !fh.FeedbackManager.IsWaitingFeedback(userID) {
		return nil
	}

	fh.FeedbackManager.CancelFeedback(userID)

	// Ответ юзеру с боагодарностью за feedback
	if err := c.Send(messages.ThanksFeedbackMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}); err != nil {
		log.Println("[ERROR] Проблема с отравкой ообщения после отзыва:", err)
	}

	msg := strings.TrimSpace(c.Text())
	if msg != "" {
		if err := fh.feedbackRepo.Create(ctx, userID, c.Sender().Username, c.Sender().FirstName, msg); err != nil {
			log.Printf("[FEEDBACK][WARN] save failed: user=%d err=%v", userID, err)
		}
	}

	for _, adminID := range fh.AdminsID {
		adminMsg := fmt.Sprintf("📬 Новый отзыв от @%s (%d):\n\n%s", c.Sender().Username, userID, msg)
		log.Println("[INFO]" + adminMsg)
		if _, err := fh.Bot.Send(&telebot.User{ID: adminID}, adminMsg); err != nil {
			log.Println("[ERROR] Проблема с отравкой ообщения после отзыва:", err)
		}
	}

	return nil
}
