package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestSessionRepo_CreateSession(t *testing.T) {
	cleanDB(t)

	repo := NewSessionRepo(testDatabase)
	ctx := context.Background()

	chatID := int64(12345)
	id, err := repo.CreateSession(ctx, chatID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected id != 0")
	}

	var gotChatID int64
	var startedAt time.Time
	var endedAt sql.NullTime
	var players int

	err = testDatabase.DB.QueryRow(`
		SELECT chat_id, started_at, ended_at, players
		FROM sessions
		WHERE id = $1
	`, id).Scan(&gotChatID, &startedAt, &endedAt, &players)
	if err != nil {
		t.Fatalf("select created session: %v", err)
	}

	if gotChatID != chatID {
		t.Fatalf("expected chat_id=%d, got %d", chatID, gotChatID)
	}
	if endedAt.Valid {
		t.Fatalf("expected ended_at NULL, got %v", endedAt.Time)
	}
	if players != 0 {
		t.Fatalf("expected players=0, got %d", players)
	}
}

func TestSessionRepo_HasAnySession(t *testing.T) {
	cleanDB(t)

	repo := NewSessionRepo(testDatabase)
	ctx := context.Background()

	chatID := int64(999)

	exists, err := repo.HasAnySession(ctx, chatID)
	if err != nil {
		t.Fatalf("HasAnySession (empty): %v", err)
	}
	if exists {
		t.Fatalf("expected exists=false")
	}

	if _, err := repo.CreateSession(ctx, chatID); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	exists, err = repo.HasAnySession(ctx, chatID)
	if err != nil {
		t.Fatalf("HasAnySession (after): %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true")
	}
}

func TestSessionRepo_IncPlayers(t *testing.T) {
	cleanDB(t)

	repo := NewSessionRepo(testDatabase)
	ctx := context.Background()

	chatID := int64(777)
	sessionID, err := repo.CreateSession(ctx, chatID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// +1
	if err := repo.IncPlayers(ctx, sessionID); err != nil {
		t.Fatalf("IncPlayers: %v", err)
	}
	// +1
	if err := repo.IncPlayers(ctx, sessionID); err != nil {
		t.Fatalf("IncPlayers 2: %v", err)
	}

	var players int
	err = testDatabase.DB.QueryRow(`SELECT players FROM sessions WHERE id=$1`, sessionID).Scan(&players)
	if err != nil {
		t.Fatalf("select players: %v", err)
	}
	if players != 2 {
		t.Fatalf("expected players=2, got %d", players)
	}

	if err := repo.IncPlayers(ctx, 0); err != nil {
		t.Fatalf("IncPlayers(0): %v", err)
	}
}

func TestSessionRepo_FinishSession(t *testing.T) {
	cleanDB(t)

	repo := NewSessionRepo(testDatabase)
	ctx := context.Background()

	sessionID, err := repo.CreateSession(ctx, 555)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var endedAt sql.NullTime
	if err := testDatabase.DB.QueryRow(`SELECT ended_at FROM sessions WHERE id=$1`, sessionID).Scan(&endedAt); err != nil {
		t.Fatalf("select ended_at before: %v", err)
	}
	if endedAt.Valid {
		t.Fatalf("expected ended_at NULL before finish")
	}

	if err := repo.FinishSession(ctx, sessionID); err != nil {
		t.Fatalf("FinishSession: %v", err)
	}

	if err := testDatabase.DB.QueryRow(`SELECT ended_at FROM sessions WHERE id=$1`, sessionID).Scan(&endedAt); err != nil {
		t.Fatalf("select ended_at after: %v", err)
	}
	if !endedAt.Valid {
		t.Fatalf("expected ended_at to be set after finish")
	}
}
