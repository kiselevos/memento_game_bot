package bot

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
	"fmt"
	"sort"
	"strings"

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
	h.Bot.Handle("/vote", h.StartVote)
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
		// Подумать о функционале, возможно заменять фото???
		return nil
	}

	h.GameManager.TakePhoto(chat.ID, user, fileID)

	return c.Send(fmt.Sprintf("%s, ваше фото принято.", session.GetUserName(user.ID)))
}

func (h *Handlers) StartVote(c telebot.Context) error {

	chat := c.Chat()

	session, exist := h.GameManager.GetSession(chat.ID)
	if !exist || session.State != game.RoundStartState {
		return c.Send("На данный момент нет запущенного раунда")
	}

	// // Для честного голосования?
	// if len(session.UsersPhoto) < 2 {
	// 	return c.Send(messages.NotEnoughPlayers)
	// }

	h.GameManager.StartVoting(session)

	// Нужно л обновлять ссесию?
	session, _ = h.GameManager.GetSession(chat.ID)

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

		h.Bot.Send(chat, &telebot.Photo{File: telebot.File{FileID: val.PhotoID}},
			&telebot.SendOptions{
				ReplyMarkup: &telebot.ReplyMarkup{InlineKeyboard: [][]telebot.InlineButton{{button}}},
			})

	}

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
	if !exist || session.State != game.VoteState {
		return c.Respond(&telebot.CallbackResponse{
			Text: "Голосование ещё не началось или уже завершено.",
		})
	}

	if _, voted := session.Votes[voter.ID]; voted {
		return c.Respond(&telebot.CallbackResponse{
			Text: "Вы уже проголосовали!",
		})
	}

	targetUserID, exists := session.IndexPhotoToUser[photoNum]
	if !exists {
		return c.Respond(&telebot.CallbackResponse{
			Text: "Ошибка: неизвестный номер фото.",
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
		Text: fmt.Sprintf("Ваш голос учтён за фото №%d!", photoNum),
	})
	if err != nil {
		return err
	}

	if len(session.Votes) == len(session.UsersPhoto) {
		h.FinishVoting(chatID, session)
	}

	return c.Send(fmt.Sprintf("%s проголосовал(а)", session.GetUserName(voter.ID)))
}

func (h *Handlers) FinishVoting(chatID int64, session *game.GameSession) {
	var result strings.Builder
	result.WriteString("🏆 Результаты голосования:\n\n")

	// Собираем результаты
	type playerResult struct {
		userID int64
		score  int
	}

	var results []playerResult
	for userID, score := range session.Score {
		results = append(results, playerResult{userID, score})
	}

	// Сортируем по убыванию голосов
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Формируем сообщение
	for i, res := range results {
		name := session.GetUserName(res.userID)
		result.WriteString(fmt.Sprintf("%d. %s — %d голосов\n", i+1, name, res.score))
	}

	session.State = game.WaitingState

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{h.startRoundBtn}}

	h.Bot.Send(&telebot.Chat{ID: chatID}, result.String(), markup)
}
