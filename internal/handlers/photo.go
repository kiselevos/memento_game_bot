package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"fmt"

	"gopkg.in/telebot.v3"
)

type PhotoHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager
	TasksList   *tasks.TasksList

	startRoundBtn telebot.InlineButton
}

func NewPhotoHandlers(bot botinterface.BotInterface, gm *game.GameManager, tl *tasks.TasksList) *PhotoHandlers {

	h := &PhotoHandlers{
		Bot:         bot,
		GameManager: gm,
		TasksList:   tl,
	}
	h.startRoundBtn = telebot.InlineButton{
		Unique: "start_round",
		Text:   "Начать раунд",
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

	// Удаляем фотографию
	_ = ph.Bot.Delete(c.Message())

	ph.GameManager.TakePhoto(chat.ID, user, fileID)

	return c.Send(fmt.Sprintf("%s, %s", session.GetUserName(user.ID), messages.PhotoReceived))
}
