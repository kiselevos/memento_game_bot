package handlers

import (
	"fmt"
	"strconv"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/bot"
	"github.com/kiselevos/memento_game_bot/internal/bot/middleware"
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/game"

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

	vh.Bot.Handle("/vote", vh.StartVote, middleware.OnlyHost(vh.GameManager))
	vh.Bot.Handle("/finishvote", vh.HandleFinishVote, middleware.OnlyHost(vh.GameManager))

	vh.Bot.Handle(&vh.StartVoteBtn, vh.StartVote, middleware.OnlyHost(vh.GameManager))
	vh.Bot.Handle(&vh.FinishVoteBtn, vh.HandleFinishVote, middleware.OnlyHost(vh.GameManager))

	// Обработчик для голосования
	vh.Bot.Handle(&telebot.InlineButton{Unique: "vote"}, vh.HandleVoteCallback)

	// для прода
	// h.Bot.Handle("/vote", GroupOnly(h.StartVote))
	// h.Bot.Handle("/finishvote", GroupOnly(h.HandleFinishVote))
}

func (vh *VoteHandlers) StartVote(c telebot.Context) error {

	chatID := c.Chat().ID

	photos, err := vh.GameManager.StartVoting(chatID)
	if err != nil {
		switch err {
		case game.ErrNoSession:
			return c.Send(messages.GameNotStarted)
		case game.ErrNoPhotosToVote:
			return c.Send(messages.NotEnoughPhoto, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
		case game.ErrFSMState:
			return c.Send("На данный момент нет запущенного раунда")
		default:
			return c.Send(messages.ErrorMessagesForUser)
		}
	}

	_ = c.Send(messages.VotingStartedMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML})

	for _, p := range photos {
		btn := telebot.InlineButton{
			Unique: "vote",
			Text:   fmt.Sprintf("Голосовать за фото №%d", p.Num),
			Data:   fmt.Sprintf("%d", p.Num),
		}

		markup := &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{{btn}},
		}

		if vh.Bot != nil {
			_, _ = vh.Bot.Send(
				&telebot.Chat{ID: chatID},
				&telebot.Photo{File: telebot.File{FileID: p.PhotoID}},
				&telebot.SendOptions{ReplyMarkup: markup},
			)
		}
	}

	// // Для честного голосования?
	// if len(session.UsersPhoto) < 2 {
	// 	return c.Send(messages.NotEnoughPlayers)
	// }

	// go vh.voteTimeout(chat.ID, session)

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{vh.FinishVoteBtn}}

	return c.Send(messages.VoitingMessage, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
}

func (vh *VoteHandlers) HandleVoteCallback(c telebot.Context) error {
	chatID := c.Chat().ID
	voter := game.GetUserFromTelebot(c.Sender())

	cb := c.Callback()
	if cb == nil {
		return nil
	}

	photoNum, err := strconv.Atoi(cb.Data)
	if err != nil {
		_ = c.Respond(&telebot.CallbackResponse{Text: messages.ErrorMessagesForUser})
		return nil
	}

	result, err := vh.GameManager.RegisterVote(chatID, &voter, photoNum)
	if err != nil {
		_ = c.Respond(&telebot.CallbackResponse{Text: messages.ErrorMessagesForUser})
		return nil
	}

	if result.IsCallback || result.IsError {
		_ = c.Respond(&telebot.CallbackResponse{Text: result.Message})
		return nil
	}

	_ = c.Respond(&telebot.CallbackResponse{Text: messages.VotedReceived})
	return c.Send(result.Message, &telebot.SendOptions{ParseMode: telebot.ModeHTML})
}

func (vh *VoteHandlers) HandleFinishVote(c telebot.Context) error {
	chatID := c.Chat().ID

	// 1) Завершаем голосование (FSM transition) внутри actor
	if err := vh.GameManager.FinishVoting(chatID); err != nil {
		switch err {
		case game.ErrNoSession:
			return c.Send(messages.GameNotStarted)
		case game.ErrFSMState:
			return c.Send("Сейчас голосование не активно.")
		default:
			return c.Send(messages.ErrorMessagesForUser)
		}
	}

	// 2) Забираем результаты раунда внутри actor
	scores, err := vh.GameManager.GetRoundScore(chatID)
	if err != nil {
		return c.Send(messages.ErrorMessagesForUser)
	}

	result := bot.RenderScore(bot.RoundScore, scores)

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{vh.RoundHandlers.StartRoundBtn}}

	if vh.Bot != nil {
		vh.Bot.Send(&telebot.Chat{ID: chatID}, result, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}
	return nil
}

// // Таймер на голосование (отключен)
// func (vh *VoteHandlers) voteTimeout(chatID int64, session *game.GameSession) {
// 	const voteDuration = 33 * time.Second

// 	time.Sleep(voteDuration)

// 	session, exist := vh.GameManager.GetSession(chatID)
// 	if !exist || session.FSM.Current() != game.VoteState {
// 		return
// 	}
// 	if vh.Bot != nil {
// 		vh.Bot.Send(&telebot.Chat{ID: chatID}, "⏳ Голосование завершено автоматически!")
// 	}
// 	log.Printf("[TIMER] Автоматическое завершение голосования в чате %d", chatID)
// 	vh.FinishVoting(chatID, session)
// }
