package handlers

import (
	messages "PhotoBattleBot/assets"
	"PhotoBattleBot/internal/bot"
	"PhotoBattleBot/internal/bot/middleware"
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/game"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

type VoteHandlers struct {
	Bot         botinterface.BotInterface
	GameManager *game.GameManager

	RoundHandlers *RoundHandlers

	StartVoteBtn  telebot.InlineButton
	FinishVoteBtn telebot.InlineButton
}

func NewVoteHandlers(bot botinterface.BotInterface, gm *game.GameManager) *VoteHandlers {

	h := &VoteHandlers{
		Bot:         bot,
		GameManager: gm,
	}

	h.StartVoteBtn = telebot.InlineButton{
		Unique: "start_vote",
		Text:   "Начать голосование",
	}
	h.FinishVoteBtn = telebot.InlineButton{
		Unique: "finish_vote",
		Text:   "Завершить голосование",
	}

	return h
}

func (vh *VoteHandlers) Register() {

	vh.Bot.Handle("/vote", vh.StartVote, middleware.OnlyAdmins(vh.Bot))
	vh.Bot.Handle("/finishvote", vh.HandleFinishVote, middleware.OnlyAdmins(vh.Bot))

	vh.Bot.Handle(&vh.StartVoteBtn, vh.StartVote, middleware.OnlyAdmins(vh.Bot))
	vh.Bot.Handle(&vh.FinishVoteBtn, vh.HandleFinishVote, middleware.OnlyAdmins(vh.Bot))

	// для прода
	// h.Bot.Handle("/vote", GroupOnly(h.StartVote))
	// h.Bot.Handle("/finishvote", GroupOnly(h.HandleFinishVote))
}

func (vh *VoteHandlers) StartVote(c telebot.Context) error {

	chat := c.Chat()

	session, exist := vh.GameManager.GetSession(chat.ID)
	if !exist || session.FSM.Current() != game.RoundStartState {
		log.Printf("[INFO] Попытка запуска голосования без раунда %d", chat.ID)
		return c.Send("На данный момент нет запущенного раунда")
	}

	// Зашита от нулевого голосования когда никто не скинул фото)
	if len(session.UsersPhoto) == 0 {
		return c.Send(messages.NotEnoughPhoto, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	}

	// // Для честного голосования?
	// if len(session.UsersPhoto) < 2 {
	// 	return c.Send(messages.NotEnoughPlayers)
	// }

	err := vh.GameManager.StartVoting(session)
	if err != nil {
		log.Printf("[INFO] Попытка запуска голосования без раунда %d", chat.ID)
		return c.Send(messages.ErrorMessagesForUser)
	}

	if err := c.Send(messages.VotingStartedMessage, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}); err != nil {
		log.Printf("[ERROR] Не удалось отправить VotingStartedMessage: %v", err)
	}

	time.Sleep(1 * time.Second)

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

		vh.Bot.Handle(&button, vh.makeVoteHandler(chat.ID, indexPhoto))
		if vh.Bot != nil {
			vh.Bot.Send(chat, &telebot.Photo{File: telebot.File{FileID: val.PhotoID}},
				&telebot.SendOptions{
					ReplyMarkup: &telebot.ReplyMarkup{InlineKeyboard: [][]telebot.InlineButton{{button}}},
				})
		}
	}

	// go vh.voteTimeout(chat.ID, session)

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{vh.FinishVoteBtn}}

	return c.Send(messages.VoitingMessage, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
}

func (vh *VoteHandlers) makeVoteHandler(chatID int64, photoNum int) func(telebot.Context) error {
	return func(c telebot.Context) error {
		return vh.HandleVote(c, chatID, photoNum)
	}
}

func (vh *VoteHandlers) HandleVote(c telebot.Context, chatID int64, photoNum int) error {

	voter := c.Sender()

	result, err := vh.GameManager.RegisterVote(chatID, voter, photoNum)
	if err != nil && result.IsCallback {
		_ = c.Respond(&telebot.CallbackResponse{Text: result.Message})
		return nil
	}

	if result.IsCallback {
		return c.Respond(&telebot.CallbackResponse{Text: result.Message})
	}

	_ = c.Respond(&telebot.CallbackResponse{Text: messages.VotedReceived})

	return c.Send(result.Message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (vh *VoteHandlers) FinishVoting(chatID int64, session *game.GameSession) {

	if session.FSM.Current() != game.VoteState {
		log.Printf("[WARN] Попытка повторного завершения голосования в чате %d", chatID)
		return
	}

	vh.GameManager.FinishVoting(session)
	result := bot.RenderScore(bot.RoundScore, session.RoundScore())

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{vh.RoundHandlers.StartRoundBtn}}

	if vh.Bot != nil {
		vh.Bot.Send(&telebot.Chat{ID: chatID}, result, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}, markup)
	}
}

func (vh *VoteHandlers) HandleFinishVote(c telebot.Context) error {
	chatID := c.Chat().ID

	session, exist := vh.GameManager.GetSession(chatID)
	if !exist || session.FSM.Current() != game.VoteState {
		log.Printf("[INFO] Попытка окончания голосования без раунда %d", chatID)
		return c.Send("Сейчас голосование не активно.")
	}

	vh.FinishVoting(chatID, session)
	return nil
}

// Таймер на голосование (отключен)
func (vh *VoteHandlers) voteTimeout(chatID int64, session *game.GameSession) {
	const voteDuration = 33 * time.Second

	time.Sleep(voteDuration)

	session, exist := vh.GameManager.GetSession(chatID)
	if !exist || session.FSM.Current() != game.VoteState {
		return
	}
	if vh.Bot != nil {
		vh.Bot.Send(&telebot.Chat{ID: chatID}, "⏳ Голосование завершено автоматически!")
	}
	log.Printf("[TIMER] Автоматическое завершение голосования в чате %d", chatID)
	vh.FinishVoting(chatID, session)
}
