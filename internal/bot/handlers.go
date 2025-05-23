package bot

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

// Handlers структура хранящая в себе bot и GameManager для роутинга
type Handlers struct {
	Bot         BotInterface // Меняем на интерфейс для моков
	GameManager *game.GameManager
	TasksList   *tasks.TasksList

	startRoundBtn telebot.InlineButton
}

// NewHandlers создание нового хендлера через контруктор
func NewHandlers(bot BotInterface, gm *game.GameManager, tl *tasks.TasksList) *Handlers {
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
	h.Bot.Handle("/start", h.Start)

	// Для прод версии
	// h.Bot.Handle("/startgame", GroupOnly(h.StartGame))
	// h.Bot.Handle(&h.startRoundBtn, GroupOnly(h.HandleStartRound))
	// h.Bot.Handle("/newround", GroupOnly(h.HandleStartRound))
	// h.Bot.Handle(telebot.OnPhoto, GroupOnly(h.TakeUserPhoto))
	// h.Bot.Handle("/vote", GroupOnly(h.StartVote))
	// h.Bot.Handle("/finishvote", GroupOnly(h.HandleFinishVote))
	// h.Bot.Handle("/endgame", GroupOnly(h.HandleEndGame))
	// h.Bot.Handle("/score", GroupOnly(h.HandleScore))

	h.Bot.Handle("/startgame", h.StartGame)
	h.Bot.Handle(&h.startRoundBtn, h.HandleStartRound)
	h.Bot.Handle("/newround", h.HandleStartRound)
	h.Bot.Handle(telebot.OnPhoto, h.TakeUserPhoto)
	h.Bot.Handle("/vote", h.StartVote)
	h.Bot.Handle("/finishvote", h.HandleFinishVote)
	h.Bot.Handle("/endgame", h.HandleEndGame)
	h.Bot.Handle("/score", h.HandleScore)
}

func (h *Handlers) Start(c telebot.Context) error {
	return c.Send(messages.WelcomeSingleMessage)
}

func (h *Handlers) StartGame(c telebot.Context) error {

	chatID := c.Chat().ID

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{h.startRoundBtn}}

	h.GameManager.StartNewGameSession(chatID)

	if h.Bot != nil {
		h.Bot.Send(&telebot.Chat{ID: chatID}, messages.WelcomeGroupMessage)
	}

	return c.Send(messages.GameRulesText, markup)
}

func (h *Handlers) HandleStartRound(c telebot.Context) error {
	//Убираем анимацию мерцания кнопки
	if c.Callback() != nil {
		_ = c.Respond(&telebot.CallbackResponse{})
	}

	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist {
		log.Printf("[INFO] Попытка запуска раунда без начала новой игры в чате %d", chatID)
		return c.Send(messages.GameNotStarted)
	}

	task, err := h.TasksList.GetRandomTask(session.UsedTasks)
	if err != nil {
		log.Printf("[INFO] Все вопросы в чате %d закончены", chatID)
		h.HandleEndGame(c) // автоматический финал
		return nil
	}

	err = h.GameManager.StartNewRound(session, task)
	if err != nil {
		log.Printf("[ERROR] Ошибка начала нового раунда %d, %v", chatID, err)
		return c.Send(messages.ErrorMessagesForUser)
	}

	text := messages.RoundStartedMessage + "\n" + task

	btn := h.startRoundBtn
	btn.Text = "🔁 Поменять задание"

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{btn}}

	return c.Send(text, markup)
}

// TakeUserPhoto - обирает фото только в уловиях запущенного раунда.
func (h *Handlers) TakeUserPhoto(c telebot.Context) error {
	chat := c.Chat()
	user := c.Sender()

	session, exist := h.GameManager.GetSession(chat.ID)
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
	_ = h.Bot.Delete(c.Message())

	h.GameManager.TakePhoto(chat.ID, user, fileID)

	return c.Send(fmt.Sprintf("%s, %s", session.GetUserName(user.ID), messages.PhotoReceived))
}

func (h *Handlers) StartVote(c telebot.Context) error {

	chat := c.Chat()

	session, exist := h.GameManager.GetSession(chat.ID)
	if !exist || session.FSM.Current() != game.RoundStartState {
		log.Printf("[INFO] Попытка запуска голосования без раунда %d", chat.ID)
		return c.Send("На данный момент нет запущенного раунда")
	}

	// // Для честного голосования?
	// if len(session.UsersPhoto) < 2 {
	// 	return c.Send(messages.NotEnoughPlayers)
	// }

	err := h.GameManager.StartVoting(session)
	if err != nil {
		log.Printf("[INFO] Попытка запуска голосования без раунда %d", chat.ID)
		return c.Send(messages.ErrorMessagesForUser)
	}

	// вспомогательная структура для вытаскивания фото
	type photoWithInd struct {
		UserID  int64
		PhotoID string
	}

	var photos []photoWithInd

	for userID, photoID := range session.UsersPhoto {
		photos = append(photos, photoWithInd{UserID: userID, PhotoID: photoID})
	}

	session.IndexPhotoToUser = make(map[int]int64)

	for id, val := range photos {
		indexPhoto := id + 1
		button := telebot.InlineButton{
			Unique: fmt.Sprintf("vote_%d", indexPhoto),
			Text:   fmt.Sprintf("Голосовать за фото №%d", indexPhoto),
		}

		session.IndexPhotoToUser[indexPhoto] = val.UserID

		h.Bot.Handle(&button, h.makeVoteHandler(chat.ID, indexPhoto))
		if h.Bot != nil {
			h.Bot.Send(chat, &telebot.Photo{File: telebot.File{FileID: val.PhotoID}},
				&telebot.SendOptions{
					ReplyMarkup: &telebot.ReplyMarkup{InlineKeyboard: [][]telebot.InlineButton{{button}}},
				})
		}
	}

	go h.voteTimeout(chat.ID, session)

	return c.Send(messages.VotingStartedMessage)
}

func (h *Handlers) makeVoteHandler(chatID int64, photoNum int) func(telebot.Context) error {
	return func(c telebot.Context) error {
		return h.HandleVote(c, chatID, photoNum)
	}
}

func (h *Handlers) HandleVote(c telebot.Context, chatID int64, photoNum int) error {

	voter := c.Sender()

	session, exist := h.GameManager.GetSession(chatID)
	if !exist || session.FSM.Current() != game.VoteState {
		return c.Respond(&telebot.CallbackResponse{
			Text: messages.VotedEarler,
		})
	}

	if _, voted := session.Votes[voter.ID]; voted {
		return c.Respond(&telebot.CallbackResponse{
			Text: messages.VotedAlready,
		})
	}

	targetUserID, exists := session.IndexPhotoToUser[photoNum]
	if !exists {
		log.Printf("[ERROR] Hеизвестный номер фото для голосования! Номер чата: %d\n", chatID)
		return c.Respond(&telebot.CallbackResponse{
			Text: "Упсс... Ошибка уже направлена разработчику. Спасибо!",
		})
	}

	// if targetUserID == voter.ID {
	// 	return c.Respond(&telebot.CallbackResponse{
	// 		Text: "За себя голосовать не честно!",
	// 	})
	// }

	session.Votes[voter.ID] = targetUserID
	session.Score[targetUserID]++

	err := c.Respond(&telebot.CallbackResponse{
		Text: messages.VotedReceived,
	})
	if err != nil {
		return err
	}

	return c.Send(fmt.Sprintf("%s проголосовал(а)", session.GetUserName(voter.ID)))
}

func (h *Handlers) FinishVoting(chatID int64, session *game.GameSession) {

	if session.FSM.Current() != game.VoteState {
		log.Printf("[WARN] Попытка повторного завершения голосования в чате %d", chatID)
		return
	}

	h.GameManager.FinishVoting(session)
	result := RenderScore(RoundScore, session.RoundScore())

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{h.startRoundBtn}}
	if h.Bot != nil {
		h.Bot.Send(&telebot.Chat{ID: chatID}, result, markup)
	}
}

func (h *Handlers) HandleFinishVote(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist || session.FSM.Current() != game.VoteState {
		log.Printf("[INFO] Попытка окончания голосования без раунда %d", chatID)
		return c.Send("Сейчас голосование не активно.")
	}

	h.FinishVoting(chatID, session)
	return nil
}

func (h *Handlers) voteTimeout(chatID int64, session *game.GameSession) {
	const voteDuration = 33 * time.Second

	time.Sleep(voteDuration)

	session, exist := h.GameManager.GetSession(chatID)
	if !exist || session.FSM.Current() != game.VoteState {
		return
	}
	if h.Bot != nil {
		h.Bot.Send(&telebot.Chat{ID: chatID}, "⏳ Голосование завершено автоматически!")
	}
	log.Printf("[TIMER] Автоматическое завершение голосования в чате %d", chatID)
	h.FinishVoting(chatID, session)
}

func (h *Handlers) HandleEndGame(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted)
	}

	result := RenderScore(FinalScore, session.TotalScore())

	h.GameManager.EndGame(chatID)

	return c.Send(result + "\n\n" + messages.FinishGameMassage)
}

func (h *Handlers) HandleScore(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := h.GameManager.GetSession(chatID)
	if !exist {
		return c.Send(messages.GameNotStarted)
	}
	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{h.startRoundBtn}}

	result := RenderScore(GameScore, session.TotalScore())
	return c.Send(result, markup)
}
