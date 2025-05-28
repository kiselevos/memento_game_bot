package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"fmt"
	"strings"

	"gopkg.in/telebot.v3"
)

type PhotoHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager
}

func NewPhotoHandlers(bot botinterface.BotInterface, gm *game.GameManager) *PhotoHandlers {

	h := &PhotoHandlers{
		Bot:         bot,
		GameManager: gm,
	}

	return h
}

func (ph *PhotoHandlers) Register() {

	ph.Bot.Handle(telebot.OnPhoto, ph.TakeUserPhoto)

	// Для прод версии
	// h.Bot.Handle(telebot.OnPhoto, GroupOnly(h.TakeUserPhoto))

}

// TakeUserPhoto - обирает фото только в уловиях запущенного раунда.
func (ph *PhotoHandlers) TakeUserPhoto(c telebot.Context) error {
	chat := c.Chat()
	user := c.Sender()

	session, exist := ph.GameManager.GetSession(chat.ID)
	if !exist || session.FSM.Current() != game.RoundStartState {
		return nil
	}

	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	fileID := photo.File.FileID

	_, exist = session.UsersPhoto[user.ID]

	if exist {
		//TODO: Подумать о функционале, возможно заменять фото???
		return nil
	}

	ph.GameManager.TakePhoto(chat.ID, user, fileID)

	// Удаляем фото только если блиц.
	if strings.HasPrefix(session.CarrentTask, "[БЛИЦ]") {
		_ = ph.Bot.Delete(c.Message())
		return c.Send(fmt.Sprintf("%s, %s", session.GetUserName(user.ID), messages.BlitsPhotoReceived))
	}

	return c.Send(fmt.Sprintf("%s, %s", session.GetUserName(user.ID), messages.PhotoReceived))
}
