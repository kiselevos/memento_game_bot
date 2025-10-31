package handlers

import (
	"log"

	messages "github.com/kiselevos/photo_battle_bot/assets"
	"github.com/kiselevos/photo_battle_bot/internal/bot/middleware"
	"github.com/kiselevos/photo_battle_bot/internal/botinterface"
	"github.com/kiselevos/photo_battle_bot/internal/game"
	"github.com/kiselevos/photo_battle_bot/internal/tasks"

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

	rh.Bot.Handle(&rh.StartRoundBtn, rh.HandleStartRound, middleware.OnlyAdmins(rh.Bot))
	rh.Bot.Handle("/newround", rh.HandleStartRound, middleware.OnlyAdmins(rh.Bot))

	// Для прод версии
	// h.Bot.Handle(&h.startRoundBtn, GroupOnly(h.HandleStartRound))
	// h.Bot.Handle("/newround", GroupOnly(h.HandleStartRound))
}

func (rh *RoundHandlers) HandleStartRound(c telebot.Context) error {

	log.Printf("Callback from: %s", c.Sender().Username)
	err := c.Respond()
	if err != nil {
		log.Printf("Respond error: %v", err)
	}
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
		return c.Send(messages.GameNotStarted, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
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
		return c.Send(messages.ErrorMessagesForUser, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
	}

	text := messages.RoundStartedMessage + "\n<b>" + task + "</b>"

	btn := rh.StartRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}
