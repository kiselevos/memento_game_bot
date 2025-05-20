package bot

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"fmt"

	"gopkg.in/telebot.v3"
)

// Handlers структура хранящая в себе bot и GameManager для роутинга
type Handlers struct {
	Bot         *telebot.Bot
	GameManager *game.GameManager
	TasksList   *tasks.TasksList

	startRoundBtn telebot.InlineButton
}

// NewHandlers создание нового хендлера через контруктор
func NewHandlers(bot *telebot.Bot, gm *game.GameManager, tl *tasks.TasksList) *Handlers {
	h := &Handlers{
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

func (h *Handlers) Register() {
	h.Bot.Handle("/startGame", h.StartGame)
	h.Bot.Handle("/start", h.Start)
	h.Bot.Handle(&h.startRoundBtn, h.OnStartRound)
	h.Bot.Handle(telebot.OnPhoto, h.TakeUserPhoto)
}

func (h *Handlers) Start(c telebot.Context) error {
	return c.Send(messages.WelcomeMessage)
}

func (h *Handlers) StartGame(c telebot.Context) error {

	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{h.startRoundBtn}}

	h.GameManager.StartNewGameSession(chatID)

	return c.Send(messages.GameRulesText, markup)
}

func (h *Handlers) OnStartRound(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted)
	}

	task, err := h.TasksList.GetRandomTask(session.UsedTasks)
	if err != nil {
		return c.Send(messages.TheEndMessages)
	}

	h.GameManager.StartNewRound(session, task)

	text := messages.RoundStartedMessage + "\n" + task

	return c.Send(text)
}

// TakeUserPhoto - обирает фото только в уловиях запущенного раунда.
func (h *Handlers) TakeUserPhoto(c telebot.Context) error {
	chat := c.Chat()
	user := c.Sender()

	session, exist := h.GameManager.GetSession(chat.ID)
	if !exist || session.State != game.RoundStartState {
		return nil
	}

	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	fileID := photo.File.FileID

	// Удаляем фотографию
	_ = h.Bot.Delete(c.Message())

	_, exist = session.UsersPhoto[user.ID]

	if exist {
		return nil
	}

	h.GameManager.TakePhoto(chat.ID, user, fileID)

	return c.Send(fmt.Sprintf("%s, ваше фото принято.", session.GetUserName(user.ID)))
}
