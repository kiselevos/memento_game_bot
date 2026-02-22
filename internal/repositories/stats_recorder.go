package repositories

import (
	"context"
	"log/slog"

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
	l := slog.Default().With("component", "recorder", "action", "increment_task_usage", "task_id", taskID)

	if countPhoto > 0 {
		if err := r.taskRepo.IncUse(ctx, taskID, countPhoto); err != nil {
			l.Error("db update failed: inc use_count", "count_photo", countPhoto, "err", err)
		}
		return
	}

	if err := r.taskRepo.IncSkip(ctx, taskID); err != nil {
		l.Error("db update failed: inc skip_count", "err", err)
	}
}

// Статы старта новой игры
func (r *Recorder) StatsForStartNewGame(ctx context.Context, user game.User, sessionID int64) {
	l := slog.Default().With("component", "recorder", "action", "stats_for_start_new_game", "user_id", user.ID)

	if err := r.userRepo.CreateIfNotExists(ctx, user.ID, user.Username, user.FirstName); err != nil {
		l.Error("db update failed: create user if not exists", "err", err)
	}

	if err := r.userRepo.IncGamesPlayed(ctx, user.ID); err != nil {
		l.Error("db update failed: inc games played", "err", err)
	}

	if err := r.sessionRepo.IncPlayers(ctx, sessionID); err != nil {
		l.Error("db update failed: session inc player", "err", err)
	}
}

func (r *Recorder) UsersVotesStatsUpdate(ctx context.Context, scores []game.PlayerScore, usersIDs []int64) {
	l := slog.Default().With("component", "recorder", "action", "users_votes_stats_update")

	votesByUser := make(map[int64]int64, len(scores))
	for _, u := range scores {
		votesByUser[u.UserID] = int64(u.Value)
	}

	if err := r.userRepo.IncUsersPhotosSent(ctx, usersIDs); err != nil {
		l.Error("db update failed: inc users photos", "users", len(usersIDs), "err", err)
	}

	if err := r.userRepo.IncUsersVotes(ctx, votesByUser); err != nil {
		l.Error("db update failed: inc users votes", "users", len(votesByUser), "err", err)
	}
}

func (r *Recorder) CreateSessionRecord(ctx context.Context, chatID int64) int64 {
	l := slog.Default().With("component", "recorder", "action", "create_session_record", "chat_id", chatID)

	id, err := r.sessionRepo.CreateSession(ctx, chatID)
	if err != nil {
		l.Error("db insert failed: create session", "err", err)
		return 0
	}

	return id
}

func (r *Recorder) FinishSessionRecord(ctx context.Context, sessionID int64) {
	l := slog.Default().With("component", "recorder", "action", "finish_session_record", "session_id", sessionID)

	if err := r.sessionRepo.FinishSession(ctx, sessionID); err != nil {
		l.Error("db update failed: finish session", "err", err)
	}
}

func (r *Recorder) IsFirstGame(ctx context.Context, chatID int64) bool {
	l := slog.Default().With("component", "recorder", "action", "is_first_game", "chat_id", chatID)

	res, err := r.sessionRepo.HasAnySession(ctx, chatID)
	if err != nil {
		l.Error("db query failed: has any session", "err", err)
		return false
	}

	return !res
}
