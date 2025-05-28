package handlers

import (
	"PhotoBattleBot/internal/botinterface"
	"PhotoBattleBot/internal/feedback"
	"PhotoBattleBot/internal/game"
	"PhotoBattleBot/internal/tasks"
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
	adminsID []int64,
	botName string,
	gm *game.GameManager,
	tl *tasks.TasksList,
) *Handlers {

	h := &Handlers{
		Game:     NewGameHandlers(bot, gm),
		Round:    NewRoundHandlers(bot, gm, tl),
		Vote:     NewVoteHandlers(bot, gm),
		Score:    NewScoreHandlers(bot, gm),
		Feedback: NewFeedbackHandler(bot, fm, adminsID, botName),
		Photo:    NewPhotoHandlers(bot, gm),
	}

	h.Round.GameHandlers = h.Game
	h.Game.FeedbackHandlers = h.Feedback

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
