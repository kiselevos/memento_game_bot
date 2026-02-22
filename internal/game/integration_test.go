package game

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/kiselevos/memento_game_bot/internal/models"
)

// --- fakes for integration ---

type itStats struct {
	createSessionID int64

	createSessionCalls int64
	finishCalls        int64
	finishedSessionID  int64

	startNewGameCalls int64
}

func (s *itStats) IncrementTaskUsage(ctx context.Context, taskID, countPhoto int64) {}
func (s *itStats) StatsForStartNewGame(ctx context.Context, user User, sessionID int64) {
	atomic.AddInt64(&s.startNewGameCalls, 1)
}
func (s *itStats) UsersVotesStatsUpdate(ctx context.Context, scores []PlayerScore, usersIDs []int64) {
}
func (s *itStats) CreateSessionRecord(ctx context.Context, chatID int64) int64 {
	atomic.AddInt64(&s.createSessionCalls, 1)
	return s.createSessionID
}
func (s *itStats) FinishSessionRecord(ctx context.Context, sessionID int64) {
	atomic.AddInt64(&s.finishCalls, 1)
	atomic.StoreInt64(&s.finishedSessionID, sessionID)
}
func (s *itStats) IsFirstGame(ctx context.Context, chatID int64) bool { return false }

type itTasks struct {
	tasks []models.Task
	err   error
}

func (ts *itTasks) GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error) {
	if ts.err != nil {
		return nil, ts.err
	}
	out := make([]models.Task, len(ts.tasks))
	copy(out, ts.tasks)
	return out, nil
}

func TestIntegration_FullGameFlow_2Players(t *testing.T) {
	chatID := int64(1001)

	stats := &itStats{createSessionID: 777}
	taskStore := &itTasks{tasks: []models.Task{
		{ID: 1, Text: "T1"},
		{ID: 2, Text: "T2"},
		{ID: 3, Text: "T3"},
	}}

	gm := NewGameManager(context.Background(), stats, taskStore)

	host := User{ID: 1, FirstName: "Host"}
	p2 := User{ID: 2, FirstName: "Alice"}

	// 1) start session
	fallbackUsed, err := gm.StartNewGameSession(chatID, host)
	if err != nil {
		t.Fatalf("StartNewGameSession error: %v", err)
	}
	if fallbackUsed {
		t.Fatalf("fallbackUsed=true, want false")
	}

	// 2) start round (goes to RoundStartState + picks task + resets round maps)
	round, taskText, err := gm.StartNewRound(chatID, host.ID)
	if err != nil {
		t.Fatalf("StartNewRound error: %v", err)
	}
	if round != 1 {
		t.Fatalf("round=%d, want 1", round)
	}
	if taskText == "" {
		t.Fatalf("taskText empty")
	}

	// 3) submit photos (allowed only in RoundStartState)
	_, replaced, err := gm.SubmitPhoto(chatID, &host, "file-host-1")
	if err != nil {
		t.Fatalf("SubmitPhoto host error: %v", err)
	}
	if replaced {
		t.Fatalf("expected replaced=false for first submit")
	}

	_, replaced, err = gm.SubmitPhoto(chatID, &p2, "file-p2-1")
	if err != nil {
		t.Fatalf("SubmitPhoto p2 error: %v", err)
	}
	if replaced {
		t.Fatalf("expected replaced=false for first submit")
	}

	// 3.1) replace photo (should return replaced=true)
	_, replaced, err = gm.SubmitPhoto(chatID, &p2, "file-p2-2")
	if err != nil {
		t.Fatalf("SubmitPhoto p2 replace error: %v", err)
	}
	if !replaced {
		t.Fatalf("expected replaced=true")
	}

	// 4) start voting (goes to VoteState + creates mapping Num->User)
	photos, err := gm.StartVoting(chatID, host.ID)
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}
	if len(photos) != 2 {
		t.Fatalf("len(photos)=%d, want 2", len(photos))
	}

	// Найдём номера фоток, чтобы голоса были "не за себя"
	var numHost, numP2 int
	for _, it := range photos {
		if it.UserID == host.ID {
			numHost = it.Num
		}
		if it.UserID == p2.ID {
			numP2 = it.Num
		}
	}
	if numHost == 0 || numP2 == 0 {
		t.Fatalf("failed to locate photo nums: host=%d p2=%d", numHost, numP2)
	}

	// 5) register votes
	// host голосует за p2, p2 голосует за host
	res, err := gm.RegisterVote(chatID, &host, numP2)
	if err != nil {
		t.Fatalf("RegisterVote host error: %v", err)
	}
	if res.IsError || res.IsCallback {
		t.Fatalf("unexpected VoteResult for host: %+v", *res)
	}

	res, err = gm.RegisterVote(chatID, &p2, numHost)
	if err != nil {
		t.Fatalf("RegisterVote p2 error: %v", err)
	}
	if res.IsError || res.IsCallback {
		t.Fatalf("unexpected VoteResult for p2: %+v", *res)
	}

	// 5.1) duplicate vote should be callback message, no error
	res, err = gm.RegisterVote(chatID, &p2, numHost)
	if err != nil {
		t.Fatalf("RegisterVote duplicate err: %v", err)
	}
	if !res.IsCallback || res.IsError {
		t.Fatalf("expected callback=true error=false on duplicate vote, got %+v", *res)
	}

	// 6) finish voting (goes to WaitingState + increments CountRounds) and returns round scores
	scores, err := gm.FinishVoting(chatID, host.ID)
	if err != nil {
		t.Fatalf("FinishVoting error: %v", err)
	}
	if len(scores) == 0 {
		t.Fatalf("expected non-empty round scores")
	}

	// 7) total score should have both players with 1 point each
	total, err := gm.GetTotalScore(chatID)
	if err != nil {
		t.Fatalf("GetTotalScore error: %v", err)
	}
	if len(total) != 2 {
		t.Fatalf("len(total)=%d, want 2", len(total))
	}

	// 8) end game clears session and writes stats
	if err := gm.EndGame(chatID, host.ID); err != nil {
		t.Fatalf("EndGame error: %v", err)
	}
	err = gm.DoWithSession(chatID, func(s *GameSession) error { return nil })
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession after EndGame, got %v", err)
	}

	if atomic.LoadInt64(&stats.createSessionCalls) != 1 {
		t.Fatalf("CreateSessionRecord calls=%d, want 1", stats.createSessionCalls)
	}
	if atomic.LoadInt64(&stats.finishCalls) != 1 {
		t.Fatalf("FinishSessionRecord calls=%d, want 1", stats.finishCalls)
	}
	if atomic.LoadInt64(&stats.finishedSessionID) != stats.createSessionID {
		t.Fatalf("finishedChatID=%d, want %d", stats.finishedSessionID, stats.createSessionID)
	}
}

func TestIntegration_VotingGuard_NoSession(t *testing.T) {
	gm := NewGameManager(context.Background(), &itStats{}, &itTasks{})
	res, err := gm.RegisterVote(999, &User{ID: 1, FirstName: "A"}, 1)
	if err != nil {
		t.Fatalf("expected err=nil, got %v", err)
	}
	if res == nil || !res.IsCallback || !res.IsError {
		t.Fatalf("expected callback+error result on ErrNoSession, got %#v", res)
	}
}

func TestIntegration_SubmitPhoto_WrongState(t *testing.T) {

	chatID := int64(2002)

	gm := NewGameManager(context.Background(), &itStats{createSessionID: 1}, &itTasks{tasks: []models.Task{{ID: 1, Text: "T"}}})
	_, err := gm.StartNewGameSession(chatID, User{ID: 1, FirstName: "Host"})
	if err != nil {
		t.Fatalf("StartNewGameSession error: %v", err)
	}

	// Не запускали round => FSM в waiting, SubmitPhoto должен запретить
	_, _, err = gm.SubmitPhoto(chatID, &User{ID: 2, FirstName: "U"}, "file")
	if !errors.Is(err, ErrRoundNotActive) {
		t.Fatalf("expected ErrRoundNotActive, got %v", err)
	}
}
