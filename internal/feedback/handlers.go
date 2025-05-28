package feedback

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/botinterface"
	"fmt"
	"log"

	"gopkg.in/telebot.v3"
)

type FeedbackHandlers struct {
	Bot             botinterface.BotInterface
	FeedbackManager *FeedbackManager
	AdminsID        []int64
	BotUsername     string
}

func NewFeedbackHandler(bot botinterface.BotInterface, fm *FeedbackManager, adminsID []int64, botName string) *FeedbackHandlers {
	return &FeedbackHandlers{
		Bot:             bot,
		FeedbackManager: fm,
		AdminsID:        adminsID,
		BotUsername:     botName,
	}
}

func (fh *FeedbackHandlers) Register() {
	fh.Bot.Handle("/feedback", fh.HandleStartFeedback)
}

func (fh *FeedbackHandlers) HandleStartFeedback(c telebot.Context) error {

	if c.Chat().Type == telebot.ChatPrivate {
		return fh.SendFeedbackInstructions(c)
	}

	btn := telebot.InlineButton{
		Text: "Оставить отзыв",
		URL:  fmt.Sprintf("https://t.me/%s?start=feedback", fh.BotUsername),
	}
	inline := &telebot.ReplyMarkup{}
	inline.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send("Спасибо! Вы можете оставить отзыв, перейдя в личные сообщения.", inline)
}

func (fh *FeedbackHandlers) SendFeedbackInstructions(c telebot.Context) error {

	userID := c.Sender().ID

	fh.FeedbackManager.StartFeedback(userID)

	cancelBtn := telebot.InlineButton{Text: "Отменить отзыв", Unique: "cancel_feedback"}
	inline := &telebot.ReplyMarkup{}
	inline.InlineKeyboard = [][]telebot.InlineButton{{cancelBtn}}

	return c.Send(messages.AboutFeedback, inline)
}

func (fh *FeedbackHandlers) HandelCancelFeedback(c telebot.Context) error {
	userID := c.Sender().ID

	fh.FeedbackManager.CancelFeedback(userID)

	return c.Edit("Отправка отзыва отменена.")
}

func (fh *FeedbackHandlers) HandelFeedbackText(c telebot.Context) error {

	userID := c.Sender().ID

	if !fh.FeedbackManager.IsWaitingFeedback(userID) {
		return nil
	}

	fh.FeedbackManager.CancelFeedback(userID)

	if err := c.Send(messages.ThanksFeedbackMessage); err != nil {
		log.Println("[ERROR] Проблема с отравкой ообщения после отзыва:", err)
	}

	for _, adminID := range fh.AdminsID {
		adminMsg := fmt.Sprintf("📬 Новый отзыв от @%s (%d):\n\n%s", c.Sender().Username, userID, c.Text())
		if _, err := fh.Bot.Send(&telebot.User{ID: adminID}, adminMsg); err != nil {
			log.Println("[ERROR] Проблема с отравкой ообщения после отзыва:", err)
		}
	}

	// TODO: логировать/сохранять в БД

	return nil

}
