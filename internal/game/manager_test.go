package game

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kiselevos/memento_game_bot/internal/models"
)

// --- fakes ---

type fakeStats struct {
	createSessionID int64
	firstGame       bool

	createSessionCalls int64

	finishSessionCalls int64
	finishedChatID     int64

	statsForStartNewGameCalls int64
	statsForStartNewGameUser  User

	incrementTaskUsageCalls int64
	incrementArgs           struct {
		taskID     int64
		countPhoto int64
	}

	usersVotesUpdateCalls int64
}

func (s *fakeStats) IncrementTaskUsage(ctx context.Context, taskID, countPhoto int64) {
	atomic.AddInt64(&s.incrementTaskUsageCalls, 1)
	s.incrementArgs.taskID = taskID
	s.incrementArgs.countPhoto = countPhoto
}

func (s *fakeStats) StatsForStartNewGame(ctx context.Context, user User, sessionID int64) {
	atomic.AddInt64(&s.statsForStartNewGameCalls, 1)
	s.statsForStartNewGameUser = user
}

func (s *fakeStats) UsersVotesStatsUpdate(ctx context.Context, scores []PlayerScore, usersIDs []int64) {
	atomic.AddInt64(&s.usersVotesUpdateCalls, 1)
}

func (s *fakeStats) CreateSessionRecord(ctx context.Context, chatID int64) int64 {
	atomic.AddInt64(&s.createSessionCalls, 1)
	return s.createSessionID
}

func (s *fakeStats) FinishSessionRecord(ctx context.Context, chatID int64) {
	atomic.AddInt64(&s.finishSessionCalls, 1)
	atomic.StoreInt64(&s.finishedChatID, chatID)
}

func (s *fakeStats) IsFirstGame(ctx context.Context, chatID int64) bool {
	return s.firstGame
}

type fakeTaskStore struct {
	tasks []models.Task
	err   error
}

func (ts *fakeTaskStore) GetActiveTaskList(ctx context.Context, category *string) ([]models.Task, error) {
	if ts.err != nil {
		return nil, ts.err
	}
	out := make([]models.Task, len(ts.tasks))
	copy(out, ts.tasks)
	return out, nil
}

// --- helpers ---

func mustStartSession(t *testing.T, gm *GameManager, chatID int64, host User) {
	t.Helper()
	_, err := gm.StartNewGameSession(chatID, host)
	if err != nil {
		t.Fatalf("StartNewGameSession() error: %v", err)
	}
}

// --- tests ---

func TestGameManager_Do_CreatesSingleActorPerChat(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	var a1, a2 *chatActor

	if err := gm.Do(100, func(a *chatActor) error {
		a1 = a
		return nil
	}); err != nil {
		t.Fatalf("Do #1 error: %v", err)
	}

	if err := gm.Do(100, func(a *chatActor) error {
		a2 = a
		return nil
	}); err != nil {
		t.Fatalf("Do #2 error: %v", err)
	}

	if a1 == nil || a2 == nil {
		t.Fatalf("actors are nil: a1=%v a2=%v", a1, a2)
	}
	if a1 != a2 {
		t.Fatalf("expected same actor pointer for same chatID")
	}
}

func TestGameManager_Do_SerializesConcurrentCalls(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	const N = 300
	var counter int // без локов - если очередь сломана, тест часто флапает

	var wg sync.WaitGroup
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = gm.Do(777, func(a *chatActor) error {
				counter++
				return nil
			})
		}()
	}

	wg.Wait()

	if counter != N {
		t.Fatalf("counter=%d, want %d (queue/actor serialization broken?)", counter, N)
	}
}

func TestGameManager_DoWithSession_NoSession(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	err := gm.DoWithSession(1, func(s *GameSession) error { return nil })
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}
}

func TestGameManager_StartNewGameSession_CreatesSessionAndHostName(t *testing.T) {
	stats := &fakeStats{createSessionID: 42}
	ts := &fakeTaskStore{
		tasks: []models.Task{
			{ID: 1, Text: "Task 1"},
			{ID: 2, Text: "Task 2"},
		},
	}
	gm := NewGameManager(context.Background(), stats, ts)

	host := User{ID: 10, FirstName: "Victoria", Username: "vic"}
	fallbackUsed, err := gm.StartNewGameSession(999, host)
	if err != nil {
		t.Fatalf("StartNewGameSession() error: %v", err)
	}
	if fallbackUsed {
		t.Fatalf("fallbackUsed=true, want false")
	}

	err = gm.DoWithSession(999, func(s *GameSession) error {
		if s.SessionID != 42 {
			t.Fatalf("SessionID=%d, want 42", s.SessionID)
		}
		if s.ChatID != 999 {
			t.Fatalf("ChatID=%d, want 999", s.ChatID)
		}
		if s.Host.ID != host.ID {
			t.Fatalf("Host.ID=%d, want %d", s.Host.ID, host.ID)
		}
		if s.FSM == nil || s.FSM.Current() != WaitingState {
			t.Fatalf("FSM invalid or not in WaitingState")
		}
		if s.UserNames == nil {
			t.Fatalf("UserNames is nil")
		}

		if len(s.Tasks) == 0 {
			t.Fatalf("Tasks not initialized")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithSession() error: %v", err)
	}

	if got := atomic.LoadInt64(&stats.createSessionCalls); got != 1 {
		t.Fatalf("CreateSessionRecord calls=%d, want 1", got)
	}
}

func TestGameManager_CheckFirstGame(t *testing.T) {
	stats := &fakeStats{firstGame: true}
	gm := NewGameManager(context.Background(), stats, &fakeTaskStore{})

	if !gm.CheckFirstGame(1) {
		t.Fatalf("expected true")
	}
}

func TestGameManager_StartNewRound_NoSession(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	_, _, err := gm.StartNewRound(111, 808080)
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}
}

func TestGameManager_SubmitPhoto_RoundNotActive(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})
	host := User{ID: 1, FirstName: "Host"}
	mustStartSession(t, gm, 123, host)

	_ = gm.DoWithSession(123, func(s *GameSession) error {
		// чтобы не сработал invariant
		if s.UsersPhoto == nil {
			s.UsersPhoto = make(map[int64]string)
		}
		s.FSM.ForceState(WaitingState) // round не активен
		return nil
	})

	_, _, err := gm.SubmitPhoto(123, &User{ID: 2, FirstName: "U"}, "file-1")
	if !errors.Is(err, ErrRoundNotActive) {
		t.Fatalf("expected ErrRoundNotActive, got %v", err)
	}
}

func TestGameManager_SubmitPhoto_InvariantUsersPhotoNil(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})
	host := User{ID: 1, FirstName: "Host"}
	mustStartSession(t, gm, 555, host)

	_ = gm.DoWithSession(555, func(s *GameSession) error {
		s.UsersPhoto = nil // ломаем инвариант
		s.FSM.ForceState(RoundStartState)
		return nil
	})

	_, _, err := gm.SubmitPhoto(555, &User{ID: 2, FirstName: "U"}, "file-1")
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("expected ErrInvariantViolation, got %v", err)
	}
}

func TestGameManager_SubmitPhoto_NewUser_UpdatesStatsOnce(t *testing.T) {
	stats := &fakeStats{createSessionID: 1}
	gm := NewGameManager(context.Background(), stats, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})
	host := User{ID: 1, FirstName: "Host"}
	mustStartSession(t, gm, 777, host)

	_ = gm.DoWithSession(777, func(s *GameSession) error {
		if s.UsersPhoto == nil {
			s.UsersPhoto = make(map[int64]string)
		}
		s.FSM.ForceState(RoundStartState)
		return nil
	})

	u := &User{ID: 2, FirstName: "Alice"}

	_, _, err := gm.SubmitPhoto(777, u, "file-1")
	if err != nil {
		t.Fatalf("SubmitPhoto() error: %v", err)
	}

	if got := atomic.LoadInt64(&stats.statsForStartNewGameCalls); got != 1 {
		t.Fatalf("StatsForStartNewGame calls=%d, want 1", got)
	}
	if stats.statsForStartNewGameUser.ID != u.ID {
		t.Fatalf("StatsForStartNewGame userID=%d, want %d", stats.statsForStartNewGameUser.ID, u.ID)
	}
}

func TestGameManager_RegisterVote_NoSession_ReturnsCallbackErrorResult(t *testing.T) {
	gm := NewGameManager(context.Background(), &fakeStats{createSessionID: 1}, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	res, err := gm.RegisterVote(999, &User{ID: 1, FirstName: "V"}, 1)
	if err != nil {
		t.Fatalf("expected err=nil, got %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil VoteResult")
	}
	if !res.IsCallback || !res.IsError {
		t.Fatalf("expected callback+error result, got %+v", *res)
	}
}

func TestGameManager_EndGame_NoSession(t *testing.T) {
	stats := &fakeStats{createSessionID: 1}
	gm := NewGameManager(context.Background(), stats, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	err := gm.EndGame(1, 808080)
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}
	if got := atomic.LoadInt64(&stats.finishSessionCalls); got != 0 {
		t.Fatalf("FinishSessionRecord calls=%d, want 0", got)
	}
}

func TestGameManager_EndGame_ClearsSession_AndWritesStats(t *testing.T) {
	stats := &fakeStats{createSessionID: 900}
	gm := NewGameManager(context.Background(), stats, &fakeTaskStore{tasks: []models.Task{{ID: 1, Text: "t"}}})

	host := User{ID: 1, FirstName: "Host"}
	mustStartSession(t, gm, 900, host)

	err := gm.EndGame(900, host.ID)
	if err != nil {
		t.Fatalf("EndGame error: %v", err)
	}

	// session cleared
	err = gm.DoWithSession(900, func(s *GameSession) error { return nil })
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("expected ErrNoSession after EndGame, got %v", err)
	}

	if got := atomic.LoadInt64(&stats.finishSessionCalls); got != 1 {
		t.Fatalf("FinishSessionRecord calls=%d, want 1", got)
	}
	if got := atomic.LoadInt64(&stats.finishedChatID); got != 900 {
		t.Fatalf("FinishSessionRecord chatID=%d, want 900", got)
	}
}
