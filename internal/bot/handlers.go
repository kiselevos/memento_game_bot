package bot

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"

	"gopkg.in/telebot.v3"
)

// Handlers структура хранящая в себе bot и GameManager для роутинга
type Handlers struct {
	Bot         *telebot.Bot
	GameManager *game.GameManager
	TasksList   *tasks.TasksList
}

// NewHandlers создание нового хендлера через контруктор
func NewHandlers(bot *telebot.Bot, gm *game.GameManager, tl *tasks.TasksList) *Handlers {
	return &Handlers{
		Bot:         bot,
		GameManager: gm,
		TasksList:   tl,
	}
}

func (h *Handlers) Register() {
	h.Bot.Handle("/startGame", h.StartGame)
	h.Bot.Handle("/start", h.Start)
}

func (h *Handlers) Start(c telebot.Context) error {
	return c.Send(messages.WelcomeMessage)
}

func (h *Handlers) StartGame(c telebot.Context) error {

	chatID := c.Chat().ID

	h.GameManager.StartNewGameSession(chatID)

	return c.Send(messages.GameRulesText)
}

func handleResetGame(c telebot.Context) error {
	return c.Send((messages.GameResetMessage))
}

func handleHelp(c telebot.Context) error {
	return c.Send(messages.HelpMessage)
}

func handleStartRound(c telebot.Context) error {
	return c.Send(messages.RoundStartedMessage)
}
