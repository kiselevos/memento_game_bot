package repositories

import (
	"context"
	"testing"

	"github.com/kiselevos/memento_game_bot/internal/game"
)

func TestRecorder_StatsForStartNewGame(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()

	r := newRecorderForTest()

	chatID := int64(2002)
	sessionID := r.CreateSessionRecord(ctx, chatID)
	if sessionID == 0 {
		t.Fatalf("session not created")
	}

	// tg_user_id (у тебя он UNIQUE), считаем что game.User.ID = tg_user_id
	u := game.User{
		ID:        55501,
		Username:  "oleg",
		FirstName: "Oleg",
	}

	r.StatsForStartNewGame(ctx, u, sessionID)

	// users.games_played должен стать 1
	var gamesPlayed int64
	err := testDatabase.QueryRow(`SELECT games_played FROM users WHERE tg_user_id=$1`, u.ID).Scan(&gamesPlayed)
	if err != nil {
		t.Fatalf("select users.games_played: %v", err)
	}
	if gamesPlayed != 1 {
		t.Fatalf("expected games_played=1, got %d", gamesPlayed)
	}

	// sessions.players должен стать 1
	var players int
	err = testDatabase.QueryRow(`SELECT players FROM sessions WHERE id=$1`, sessionID).Scan(&players)
	if err != nil {
		t.Fatalf("select sessions.players: %v", err)
	}
	if players != 1 {
		t.Fatalf("expected players=1, got %d", players)
	}

	// Повторный старт игры тем же юзером: games_played инкрементится, user не дублируется
	r.StatsForStartNewGame(ctx, u, sessionID)

	err = testDatabase.QueryRow(`SELECT games_played FROM users WHERE tg_user_id=$1`, u.ID).Scan(&gamesPlayed)
	if err != nil {
		t.Fatalf("select games_played again: %v", err)
	}
	if gamesPlayed != 2 {
		t.Fatalf("expected games_played=2, got %d", gamesPlayed)
	}
}
