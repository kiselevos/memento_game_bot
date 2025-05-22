package game

import (
	"sync"
	"testing"
)

const (
	chatID    = 888
	NewGameID = 999
)

func newTestGameManager() *GameManager {
	return &GameManager{
		sessions: map[int64]*GameSession{chatID: newTestGameSession()},
		mu:       sync.Mutex{},
	}
}

func TestGetSession(t *testing.T) {

	gm := newTestGameManager()

	t.Run("Session exists", func(t *testing.T) {
		got, exist := gm.GetSession(chatID)
		if !exist {
			t.Fatal("Expected session to exist, but it does not")
		}
		if got.ChatID != chatID {
			t.Errorf("Expected ChatID %d, got %d", chatID, got.ChatID)
		}
	})

	t.Run("Session not exists", func(t *testing.T) {
		_, exist := gm.GetSession(123456)
		if exist {
			t.Error("Expected session to not exist, but it does")
		}
	})
}

func TestStartNewSession(t *testing.T) {

	gm := newTestGameManager()

	s := gm.StartNewGameSession(NewGameID)

	if s.ChatID != NewGameID {
		t.Errorf("Expected %d, got %d", NewGameID, s.ChatID)
	}
}

func TestEndGame(t *testing.T) {
	gm := newTestGameManager()

	gm.EndGame(chatID)

	_, exist := gm.GetSession(chatID)
	if exist {
		t.Errorf("Expected session %d to be deleted", chatID)
	}
}

func TestGameManager_ConcurrentAccess(t *testing.T) {
	gm := newTestGameManager()

	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func(id int64) {
			defer wg.Done()
			gm.StartNewGameSession(id)
			s, exist := gm.GetSession(id)
			if !exist || s.ChatID != id {
				t.Errorf("Session mismatch or missing for id %d", id)
			}
		}(int64(i + 1000))
	}

	wg.Wait()
}
