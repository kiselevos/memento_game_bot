package handlers

import (
	"fmt"
	"log"
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

	msg, err := vh.Bot.Send(
		c.Chat(),
		messages.VotingStartedMessage,
		&telebot.SendOptions{ParseMode: telebot.ModeHTML},
	)
	if err != nil {
		log.Printf("[WARN] cannot send VotingStartedMessage: %v", err)
	} else if msg != nil {
		if e := vh.GameManager.SaveSystemMsgID(chatID, msg.ID); e != nil {
			log.Printf("[WARN] cannot save system msg id: %v", e)
		}
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
				log.Printf("[WARN] cannot send vote photo: %v", err)
				continue
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
		log.Printf("[WARN] cannot send vote public msg: %v", err)
		return nil
	}

	if msg != nil {
		if e := vh.GameManager.SaveSystemMsgID(chatID, msg.ID); e != nil {
			log.Printf("[WARN] cannot save system msg id (vote msg): %v", e)
		}
	}
	return nil
}

func (vh *VoteHandlers) HandleFinishVote(c telebot.Context) error {
	if c.Callback() != nil {
		_ = c.Respond()
		_ = c.Delete() // удаляем кнопку "Завершить голосование"
	}

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

	// CleanUP systemd message
	cleanID, err := vh.GameManager.PopMsgIDs(chatID)
	if err == nil {
		vh.cleanupRoundArtifacts(chatID, cleanID)
	} else {
		log.Printf("[CLEANUP][WARN] PopMsgIDs failed: chat=%d err=%v", chatID, err)
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

func (vh *VoteHandlers) cleanupRoundArtifacts(chatID int64, cleanID game.CleanupIDs) {
	if vh.Bot == nil {
		log.Printf("[CLEANUP] bot is nil, skip")
		return
	}

	empty := &telebot.ReplyMarkup{}

	// 1) Убираем inline-кнопки с фото (EditReplyMarkup)
	okEdits := 0
	for _, id := range cleanID.VotePhotoMsgIDs {
		m := &telebot.Message{
			ID:   id,
			Chat: &telebot.Chat{ID: chatID},
		}

		if _, err := vh.Bot.EditReplyMarkup(m, empty); err != nil {
			log.Printf("[CLEANUP][WARN] EditReplyMarkup failed: chat=%d msg=%d err=%v", chatID, id, err)
			continue
		}
		okEdits++
	}

	// 2) Удаляем системные сообщения (Delete)
	okDeletes := 0
	for _, id := range cleanID.SystemMsgIDs {
		m := &telebot.Message{
			ID:   id,
			Chat: &telebot.Chat{ID: chatID},
		}

		if err := vh.Bot.Delete(m); err != nil {
			log.Printf("[CLEANUP][WARN] Delete failed: chat=%d msg=%d err=%v", chatID, id, err)
			continue
		}
		okDeletes++
	}

	log.Printf(
		"[CLEANUP] chat=%d votePhotoIDs=%d edited=%d systemIDs=%d deleted=%d",
		chatID, len(cleanID.VotePhotoMsgIDs), okEdits, len(cleanID.SystemMsgIDs), okDeletes,
	)
}
