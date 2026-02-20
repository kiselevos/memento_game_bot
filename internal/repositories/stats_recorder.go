package repositories

import (
	"context"
	"log"

	"github.com/kiselevos/memento_game_bot/internal/game"
)

type Recorder struct {
	userRepo    *UserRepo
	taskRepo    *TaskRepo
	sessionRepo *SessionRepo
}

func NewRecorder(ur *UserRepo, tr *TaskRepo, sr *SessionRepo) *Recorder {
	return &Recorder{
		userRepo:    ur,
		taskRepo:    tr,
		sessionRepo: sr,
	}
}

// Инкрементируем задания
func (r *Recorder) IncrementTaskUsage(ctx context.Context, taskID, countPhoto int64) {

	if countPhoto > 0 {
		if err := r.taskRepo.IncUse(ctx, taskID, countPhoto); err != nil {
			log.Printf("[DB ERROR] add use_count task %d: %v", taskID, err)
		}
		return
	}
	if err := r.taskRepo.IncSkip(ctx, taskID); err != nil {
		log.Printf("[DB ERROR] add skip_count task %d: %v", taskID, err)
	}
}

func (r *Recorder) StatsForStartNewGame(ctx context.Context, user game.User) {

	err := r.userRepo.CreateIfNotExists(ctx, user.ID, user.Username, user.FirstName)
	if err != nil {
		log.Printf("[DB ERROR] with ctreate User: %v", err)
	}

	err = r.userRepo.IncGamesPlayed(ctx, user.ID)
	if err != nil {
		log.Printf("[DB ERROR] with add game for User: %v", err)
	}
}

func (r *Recorder) UsersVotesStatsUpdate(ctx context.Context, scores []game.PlayerScore) {

	votesByUser := make(map[int64]int64, len(scores))
	for _, u := range scores {
		votesByUser[u.UserID] = int64(u.Value)
	}

	if err := r.userRepo.IncUsersVotes(ctx, votesByUser); err != nil {
		log.Println(err)
	}
}

func (r *Recorder) CreateSessionRecord(ctx context.Context, chatID int64) int64 {
	id, err := r.sessionRepo.CreateSession(ctx, chatID)
	if err != nil {
		log.Println(err)
	}

	return id
}

func (r *Recorder) FinishSessionRecord(ctx context.Context, chatID int64) {
	err := r.sessionRepo.FinishSession(ctx, chatID)
	if err != nil {
		log.Println(err)
	}
}

func (r *Recorder) IsFirstGame(ctx context.Context, chatID int64) bool {
	res, err := r.sessionRepo.HasAnySession(ctx, chatID)
	if err != nil {
		log.Println(err)
	}
	return res
}
