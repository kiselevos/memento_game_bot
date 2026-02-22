package handlers

import (
	"errors"
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

	photos, err := vh.GameManager.StartVoting(chatID, c.Sender().ID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrNoSession):
			return c.Send(messages.GameNotStarted)

		case errors.Is(err, game.ErrNoPhotosToVote):
			return c.Send(messages.NotEnoughPhoto, &telebot.SendOptions{ParseMode: telebot.ModeHTML})

		case errors.Is(err, game.ErrWrongState):
			return c.Send(messages.RoundNotActive)

		case errors.Is(err, game.ErrOnlyHost):
			text := fmt.Sprintf(messages.OnlyHostRules)
			if c.Callback() != nil {
				_ = c.Respond(&telebot.CallbackResponse{
					Text: text,
				})
				return nil
			}
			return c.Reply(text)

		default:
			_ = c.Send(messages.ErrorMessagesForUser)
			return err
		}
	}

	msg, err := vh.Bot.Send(
		c.Chat(),
		messages.VotingStartedMessage,
		&telebot.SendOptions{ParseMode: telebot.ModeHTML},
	)

	if err != nil {
		return err
	}

	if msg != nil {
		vh.GameManager.SaveSystemMsgID(chatID, msg.ID)
	}

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
			msg, err := vh.Bot.Send(
				&telebot.Chat{ID: chatID},
				&telebot.Photo{File: telebot.File{FileID: p.PhotoID}},
				&telebot.SendOptions{ReplyMarkup: markup},
			)

			if err != nil {
				return err
			}

			if msg != nil {
				// сохраняем msg.ID в session
				_ = vh.GameManager.SaveVotePhotoMsgID(chatID, p.Num, msg.ID)
			}
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

	if vh.Bot == nil {
		return nil
	}

	msg, err := vh.Bot.Send(
		&telebot.Chat{ID: chatID},
		result.Message,
		&telebot.SendOptions{ParseMode: telebot.ModeHTML},
	)
	if err != nil {
		return err
	}

	if msg != nil {
		vh.GameManager.SaveSystemMsgID(chatID, msg.ID)
	}
	return nil
}

func (vh *VoteHandlers) HandleFinishVote(c telebot.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
		_ = c.Delete() // удаляем кнопку "Завершить голосование"
	}

	chatID := c.Chat().ID

	scores, err := vh.GameManager.FinishVoting(chatID, c.Sender().ID)
	if err != nil {
		switch {
		case errors.Is(err, game.ErrNoSession):
			return c.Send(messages.GameNotStarted)

		case errors.Is(err, game.ErrWrongState):
			return c.Send(messages.VotedNotActive)

		case errors.Is(err, game.ErrOnlyHost):
			text := fmt.Sprintf(messages.OnlyHostRules)
			if c.Callback() != nil {
				_ = c.Respond(&telebot.CallbackResponse{
					Text: text,
				})
				return nil
			}
			return c.Reply(text)

		default:
			_ = c.Send(messages.ErrorMessagesForUser)
			return err
		}
	}

	// CleanUP systemd message
	cleanID, err := vh.GameManager.PopMsgIDs(chatID)
	if err == nil {
		vh.cleanupRoundArtifacts(chatID, cleanID)
	}

	result := messages.EmptyVotedResult
	if scores != nil {
		result = bot.RenderScore(bot.RoundScore, scores)
	}

	markup := &telebot.ReplyMarkup{}
	markup.InlineKeyboard = [][]telebot.InlineButton{{vh.RoundHandlers.StartRoundBtn}}

	if vh.Bot != nil {
		vh.Bot.Send(&telebot.Chat{ID: chatID}, result, &telebot.SendOptions{ParseMode: telebot.ModeHTML}, markup)
	}

	return nil
}

func (vh *VoteHandlers) cleanupRoundArtifacts(chatID int64, cleanID game.CleanupIDs) {
	if vh.Bot == nil {
		return
	}

	empty := &telebot.ReplyMarkup{}

	// 1) Убираем inline-кнопки с фото (EditReplyMarkup)
	for _, id := range cleanID.VotePhotoMsgIDs {
		m := &telebot.Message{
			ID:   id,
			Chat: &telebot.Chat{ID: chatID},
		}

		vh.Bot.EditReplyMarkup(m, empty)
	}

	// 2) Удаляем системные сообщения (Delete)
	for _, id := range cleanID.SystemMsgIDs {
		m := &telebot.Message{
			ID:   id,
			Chat: &telebot.Chat{ID: chatID},
		}

		vh.Bot.Delete(m)
	}
}
