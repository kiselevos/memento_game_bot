package game

import (
	"context"
	"sync"
	"testing"
)

const (
	chatID = int64(888)
)

func newTestGameManager() *GameManager {
	return NewGameManager(NoopStatsRecorder{}, NoopTaskStore{})
}

func seedSession(t *testing.T, gm *GameManager, id int64) {
	t.Helper()
	if err := gm.StartNewGameSession(context.Background(), id, User{ID: 1}); err != nil {
		t.Fatalf("StartNewGameSession error: %v", err)
	}
}

func seedRoundWithOnePhoto(t *testing.T, gm *GameManager, id int64) {
	t.Helper()

	err := gm.DoWithSession(id, func(s *GameSession) error {
		// запускаем раунд через доменный метод (правильно создаст UsersPhoto)
		_, _, _, err := s.StartNewRound()
		if err != nil {
			return err
		}

		// добавляем фото
		u := &User{ID: 42, FirstName: "Tester"}
		s.TakePhoto(u, "photo_file_id_1")

		return nil
	})
	if err != nil {
		t.Fatalf("seedRoundWithOnePhoto error: %v", err)
	}
}

func TestDoWithSession_NoSession(t *testing.T) {
	gm := newTestGameManager()

	err := gm.DoWithSession(chatID, func(s *GameSession) error { return nil })
	if err != ErrNoSession {
		t.Fatalf("expected ErrNoSession, got %v", err)
	}
}

func TestStartNewGameSession_CreatesSession(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)

	var (
		gotChatID int64
		state     State
	)
	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		gotChatID = s.ChatID
		state = s.FSM.Current()
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithSession error: %v", err)
	}
	if gotChatID != chatID {
		t.Fatalf("expected ChatID %d, got %d", chatID, gotChatID)
	}
	if state != WaitingState {
		t.Fatalf("expected initial state %s, got %s", WaitingState, state)
	}
}

func TestEndGame_RemovesSession(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)

	if err := gm.EndGame(context.Background(), chatID); err != nil {
		t.Fatalf("EndGame error: %v", err)
	}

	err := gm.DoWithSession(chatID, func(s *GameSession) error { return nil })
	if err != ErrNoSession {
		t.Fatalf("expected ErrNoSession after EndGame, got %v", err)
	}
}

func TestConcurrentAccess_StartAndReadSessions(t *testing.T) {
	gm := newTestGameManager()

	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func(id int64) {
			defer wg.Done()

			if err := gm.StartNewGameSession(context.Background(), id, User{ID: 1}); err != nil {
				t.Errorf("StartNewGameSession(%d) error: %v", id, err)
				return
			}

			var got int64
			err := gm.DoWithSession(id, func(s *GameSession) error {
				got = s.ChatID
				return nil
			})
			if err != nil {
				t.Errorf("DoWithSession(%d) error: %v", id, err)
				return
			}
			if got != id {
				t.Errorf("expected ChatID %d, got %d", id, got)
			}
		}(int64(i + 1000))
	}

	wg.Wait()
}

func TestStartVoting_Success(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)
	seedRoundWithOnePhoto(t, gm, chatID)

	photos, err := gm.StartVoting(chatID)
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo item, got %d", len(photos))
	}
	if photos[0].Num != 1 {
		t.Fatalf("expected photo Num=1, got %d", photos[0].Num)
	}
	if photos[0].PhotoID != "photo_file_id_1" {
		t.Fatalf("unexpected PhotoID: %s", photos[0].PhotoID)
	}

	var (
		state    State
		votesLen int
		indexLen int
	)
	err = gm.DoWithSession(chatID, func(s *GameSession) error {
		state = s.FSM.Current()
		votesLen = len(s.Votes)
		indexLen = len(s.IndexPhotoToUser)
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithSession error: %v", err)
	}
	if state != VoteState {
		t.Fatalf("expected FSM %s, got %s", VoteState, state)
	}
	if votesLen != 0 {
		t.Fatalf("expected empty Votes at start, got %d", votesLen)
	}
	if indexLen != 1 {
		t.Fatalf("expected IndexPhotoToUser size 1, got %d", indexLen)
	}
}

func TestStartVoting_FailsOnInvalidFSMState(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)
	seedRoundWithOnePhoto(t, gm, chatID)

	// специально ломаем state: фото есть, но state = waiting
	err := gm.DoWithSession(chatID, func(s *GameSession) error {
		s.FSM.ForceState(WaitingState)
		return nil
	})
	if err != nil {
		t.Fatalf("ForceState error: %v", err)
	}

	_, err = gm.StartVoting(chatID)
	if err != ErrFSMState {
		t.Fatalf("expected ErrFSMState, got %v", err)
	}
}

func TestFinishVoting_MovesToWaiting(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)
	seedRoundWithOnePhoto(t, gm, chatID)

	_, err := gm.StartVoting(chatID)
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}

	if _, err := gm.FinishVoting(context.Background(), chatID); err != nil {
		t.Fatalf("FinishVoting error: %v", err)
	}

	var state State
	err = gm.DoWithSession(chatID, func(s *GameSession) error {
		state = s.FSM.Current()
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithSession error: %v", err)
	}
	if state != WaitingState {
		t.Fatalf("expected FSM %s, got %s", WaitingState, state)
	}
}

func TestRegisterVote_SuccessAndBlocksDoubleVoteAndSelfVote(t *testing.T) {
	gm := newTestGameManager()
	seedSession(t, gm, chatID)
	seedRoundWithOnePhoto(t, gm, chatID)

	_, err := gm.StartVoting(chatID)
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}

	voter := &User{ID: 7, FirstName: "Voter"}

	// голосуем за фото №1 (это user 42)
	res, err := gm.RegisterVote(chatID, voter, 1)
	if err != nil {
		t.Fatalf("RegisterVote error: %v", err)
	}
	if res.IsCallback {
		t.Fatalf("expected non-callback message on success, got callback: %+v", res)
	}

	// повторный голос — должен быть callback
	res, err = gm.RegisterVote(chatID, voter, 1)
	if err != nil {
		t.Fatalf("RegisterVote error: %v", err)
	}
	if !res.IsCallback {
		t.Fatalf("expected callback on double vote, got: %+v", res)
	}

	// self-vote: нужно чтобы voter был в IndexPhotoToUser. Сделаем второй раунд, добавим фото от voter.
	err = gm.DoWithSession(chatID, func(s *GameSession) error {
		_, _, _, e := s.StartNewRound()
		if e != nil {
			return e
		}
		s.TakePhoto(voter, "photo_file_id_self")
		return nil
	})
	if err != nil {
		t.Fatalf("seed self photo error: %v", err)
	}

	_, err = gm.StartVoting(chatID)
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}

	// найдём номер фото voter'а
	var selfNum int
	err = gm.DoWithSession(chatID, func(s *GameSession) error {
		for num, uid := range s.IndexPhotoToUser {
			if uid == voter.ID {
				selfNum = num
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("find self num error: %v", err)
	}
	if selfNum == 0 {
		t.Fatalf("expected to find self photo num")
	}

	res, err = gm.RegisterVote(chatID, voter, selfNum)
	if err != nil {
		t.Fatalf("RegisterVote error: %v", err)
	}
	if !res.IsCallback {
		t.Fatalf("expected callback on self-vote, got: %+v", res)
	}
}
