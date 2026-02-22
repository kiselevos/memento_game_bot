package handlers

import (
	"fmt"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

	"gopkg.in/telebot.v3"
)

type PhotoHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	VoteHandlers *VoteHandlers
}

func NewPhotoHandlers(bot botinterface.BotInterface, gm *game.GameManager) *PhotoHandlers {

	ph := &PhotoHandlers{
		Bot:         bot,
		GameManager: gm,
	}

	return ph
}

func (ph *PhotoHandlers) Register() {

	ph.Bot.Handle(telebot.OnPhoto, ph.TakeUserPhoto)

	// Для прод версии
	// h.Bot.Handle(telebot.OnPhoto, GroupOnly(h.TakeUserPhoto))

}

// TakeUserPhoto - обирает фото только в уловиях запущенного раунда.
func (ph *PhotoHandlers) TakeUserPhoto(c telebot.Context) error {
	chatID := c.Chat().ID
	user := game.GetUserFromTelebot(c.Sender())

	photo := c.Message().Photo
	if photo == nil {
		return nil
	}
	fileID := photo.File.FileID

	userName, replaced, err := ph.GameManager.SubmitPhoto(chatID, &user, fileID)
	if err != nil {
		switch err {
		case game.ErrNoSession:
			return nil
		case game.ErrRoundNotActive:
			return nil
		default:
			return nil
		}
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{ph.VoteHandlers.StartVoteBtn}}

	// Принимаем и удаялем фото
	_ = ph.Bot.Delete(c.Message())

	text := messages.PhotoReceived
	if replaced {
		text = messages.PhotoUpdate
	}

	msg, err := ph.Bot.Send(
		c.Chat(),
		fmt.Sprintf("<b>%s</b>, %s", userName, text),
		&telebot.SendOptions{ParseMode: telebot.ModeHTML},
		markup,
	)

	if err == nil && msg != nil {
		ph.GameManager.SaveSystemMsgID(chatID, msg.ID)
	}

	return nil
}
