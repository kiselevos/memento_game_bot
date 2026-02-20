package handlers

import (
	"github.com/kiselevos/memento_game_bot/internal/botinterface"
	"github.com/kiselevos/memento_game_bot/internal/feedback"
	"github.com/kiselevos/memento_game_bot/internal/game"
	"github.com/kiselevos/memento_game_bot/internal/repositories"

	"gopkg.in/telebot.v3"
)

type Handlers struct {
	Game     *GameHandlers
	Vote     *VoteHandlers
	Score    *ScoreHandlers
	Feedback *FeedbackHandlers
	Round    *RoundHandlers
	Photo    *PhotoHandlers
}

func NewHandlers(
	bot botinterface.BotInterface,
	fm *feedback.FeedbackManager,
	fr *repositories.FeedbackRepo,
	adminsID []int64,
	botInfo *telebot.User,
	gm *game.GameManager,
) *Handlers {

	h := &Handlers{
		Game:     NewGameHandlers(bot, gm, botInfo),
		Round:    NewRoundHandlers(bot, gm),
		Vote:     NewVoteHandlers(bot, gm),
		Score:    NewScoreHandlers(bot, gm),
		Feedback: NewFeedbackHandler(bot, fm, adminsID, botInfo.Username, fr),
		Photo:    NewPhotoHandlers(bot, gm),
	}

	h.Round.GameHandlers = h.Game
	h.Game.FeedbackHandlers = h.Feedback
	h.Game.RoundHandlers = h.Round
	h.Photo.VoteHandlers = h.Vote
	h.Score.RoundHandlers = h.Round
	h.Score.GameHandlers = h.Game
	h.Vote.RoundHandlers = h.Round

	return h
}

func (h *Handlers) RegisterAll() {
	h.Game.Register()
	h.Vote.Register()
	h.Score.Register()
	h.Feedback.Register()
	h.Round.Register()
	h.Photo.Register()
}
