package game

import (
	"reflect"
	"sync"
	"testing"
)

const (
	chat       = 888
	userID_1   = 255257049
	userName_1 = "kiselevos"
	userID_2   = 99999999
	userName_2 = "kolya"
	userID_3   = 11111111
	userName_3 = "maria"
)

func newTestGameSession() *GameSession {
	return &GameSession{
		ChatID:           chat,
		Score:            map[int64]int{userID_1: 2, userID_2: 5, userID_3: 0},
		UsedTasks:        make(map[string]bool),
		UserNames:        map[int64]string{userID_1: userName_1, userID_2: userName_2, userID_3: userName_3},
		Votes:            make(map[int64]int64),
		UsersPhoto:       make(map[int64]string),
		CarrentTask:      "Задание",
		IndexPhotoToUser: make(map[int]int64),
		mu:               sync.Mutex{},
	}
}

func TestGetUserNameSucces(t *testing.T) {

	s := newTestGameSession()

	t.Run("Known user", func(t *testing.T) {
		if got := s.GetUserName(userID_1); got != userName_1 {
			t.Errorf("Expected %s, got %s", userName_1, got)
		}
	})

	t.Run("Unknown user", func(t *testing.T) {
		if got := s.GetUserName(999); got != "Анонимный Осётр" {
			t.Errorf("Expected 'Анонимный Осётр', got %s", got)
		}
	})
}

var scoreResult = []PlayerScore{{userID_2, userName_2, 5}, {userID_1, userName_1, 2}, {userID_3, userName_3, 0}}

func TestRoundResult(t *testing.T) {
	s := newTestGameSession()
	s.Votes[111] = 222
	s.Votes[333] = 222
	s.UserNames[222] = "Игрок222"

	results := s.RoundScore()

	if len(results) != 1 {
		t.Fatalf("Expected 1 user in round results, got %d", len(results))
	}
	if results[0].UserID != 222 || results[0].Value != 2 {
		t.Errorf("Expected user 222 with 2 votes, got %+v", results[0])
	}
}

func TestFinalScore(t *testing.T) {
	s := newTestGameSession()

	got := s.TotalScore()

	if !reflect.DeepEqual(got, scoreResult) {
		t.Errorf("Expected final score %v, got %v", scoreResult, got)
	}
}
