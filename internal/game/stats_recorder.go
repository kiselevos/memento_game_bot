package game

import (
	"context"
)

type StatsRecorder interface {
	IncrementTaskUsage(ctx context.Context, taskID, countPhoto int64)
	StatsForStartNewGame(ctx context.Context, user User, sessionID int64)
	UsersVotesStatsUpdate(ctx context.Context, scores []PlayerScore, usersIDs []int64)
	CreateSessionRecord(ctx context.Context, chatID int64) int64
	FinishSessionRecord(ctx context.Context, chatID int64)
	IsFirstGame(ctx context.Context, chatID int64) bool
}

// NoopStatsRecorder - дефолт: ничего не делает.
type NoopStatsRecorder struct{}

func (NoopStatsRecorder) IncrementTaskUsage(ctx context.Context, taskID, countPhoto int64)     {}
func (NoopStatsRecorder) StatsForStartNewGame(ctx context.Context, user User, sessionID int64) {}
func (NoopStatsRecorder) UsersVotesStatsUpdate(ctx context.Context, scores []PlayerScore, usersIDs []int64) {
}
func (NoopStatsRecorder) CreateSessionRecord(ctx context.Context, chatID int64) int64 { return 0 }
func (NoopStatsRecorder) FinishSessionRecord(ctx context.Context, chatID int64)       {}
func (NoopStatsRecorder) IsFirstGame(ctx context.Context, chatID int64) bool          { return false }
