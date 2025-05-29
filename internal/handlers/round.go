package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"log"

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

	rh.Bot.Handle(&rh.StartRoundBtn, rh.HandleStartRound)
	rh.Bot.Handle("/newround", rh.HandleStartRound)

	// Для прод версии
	// h.Bot.Handle(&h.startRoundBtn, GroupOnly(h.HandleStartRound))
	// h.Bot.Handle("/newround", GroupOnly(h.HandleStartRound))
}

func (rh *RoundHandlers) HandleStartRound(c telebot.Context) error {
	//Убираем анимацию мерцания кнопки
	if c.Callback() != nil {
		_ = c.Respond(&telebot.CallbackResponse{})
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{rh.GameHandlers.StartGameBtn}}

	chatID := c.Chat().ID

	session, exist := rh.GameManager.GetSession(chatID)
	if !exist {
		log.Printf("[INFO] Попытка запуска раунда без начала новой игры в чате %d", chatID)
		return c.Send(messages.GameNotStarted, markup)
	}

	task, err := rh.TasksList.GetRandomTask(session.UsedTasks)
	if err != nil {
		log.Printf("[INFO] Все вопросы в чате %d закончены", chatID)
		rh.GameHandlers.HandleEndGame(c) // автоматический финал
		return nil
	}

	err = rh.GameManager.StartNewRound(session, task)
	if err != nil {
		log.Printf("[ERROR] Ошибка начала нового раунда %d, %v", chatID, err)
		return c.Send(messages.ErrorMessagesForUser)
	}

	text := messages.RoundStartedMessage + "\n" + "***" + task + "***"

	btn := rh.StartRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
}
