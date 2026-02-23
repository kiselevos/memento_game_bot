package repositories

import (
	"context"
	"testing"
)

func TestUserRepo_CreateIfNotExists(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepo(testDatabase)

	tgID := int64(1001)

	if err := repo.CreateIfNotExists(ctx, tgID, "u", "F"); err != nil {
		t.Fatalf("CreateIfNotExists: %v", err)
	}
	if err := repo.CreateIfNotExists(ctx, tgID, "u2", "F2"); err != nil {
		t.Fatalf("CreateIfNotExists again: %v", err)
	}

	var cnt int
	if err := testDatabase.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE tg_user_id=$1`, tgID).Scan(&cnt); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 user row, got %d", cnt)
	}
}

func TestUserRepo_IncGamesPlayed(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepo(testDatabase)

	tgID := int64(2002)

	if err := repo.IncGamesPlayed(ctx, tgID); err == nil {
		t.Fatalf("expected error when user does not exist")
	}

	if err := repo.CreateIfNotExists(ctx, tgID, "", ""); err != nil {
		t.Fatalf("CreateIfNotExists: %v", err)
	}

	if err := repo.IncGamesPlayed(ctx, tgID); err != nil {
		t.Fatalf("IncGamesPlayed: %v", err)
	}
	if err := repo.IncGamesPlayed(ctx, tgID); err != nil {
		t.Fatalf("IncGamesPlayed 2: %v", err)
	}

	var games int64
	if err := testDatabase.DB.QueryRow(`SELECT games_played FROM users WHERE tg_user_id=$1`, tgID).Scan(&games); err != nil {
		t.Fatalf("select games_played: %v", err)
	}
	if games != 2 {
		t.Fatalf("expected games_played=2, got %d", games)
	}
}

func TestUserRepo_IncUsersPhotosSent(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepo(testDatabase)

	if err := repo.CreateIfNotExists(ctx, 1, "u1", "U1"); err != nil {
		t.Fatalf("CreateIfNotExists u1: %v", err)
	}
	if err := repo.CreateIfNotExists(ctx, 2, "u2", "U2"); err != nil {
		t.Fatalf("CreateIfNotExists u2: %v", err)
	}

	if err := repo.IncUsersPhotosSent(ctx, []int64{1, 2}); err != nil {
		t.Fatalf("IncUsersPhotosSent: %v", err)
	}

	var p1, p2 int64
	if err := testDatabase.DB.QueryRow(`SELECT photos_sent FROM users WHERE tg_user_id=1`).Scan(&p1); err != nil {
		t.Fatalf("select photos_sent u1: %v", err)
	}
	if err := testDatabase.DB.QueryRow(`SELECT photos_sent FROM users WHERE tg_user_id=2`).Scan(&p2); err != nil {
		t.Fatalf("select photos_sent u2: %v", err)
	}

	if p1 != 1 || p2 != 1 {
		t.Fatalf("expected photos_sent=1 for both, got u1=%d u2=%d", p1, p2)
	}
}

func TestUserRepo_IncUsersVotes(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewUserRepo(testDatabase)

	if err := repo.CreateIfNotExists(ctx, 10, "u10", "U10"); err != nil {
		t.Fatalf("CreateIfNotExists u10: %v", err)
	}
	if err := repo.CreateIfNotExists(ctx, 20, "u20", "U20"); err != nil {
		t.Fatalf("CreateIfNotExists u20: %v", err)
	}

	votes := map[int64]int64{10: 5, 20: 2}
	if err := repo.IncUsersVotes(ctx, votes); err != nil {
		t.Fatalf("IncUsersVotes: %v", err)
	}

	var v10, v20 int64
	if err := testDatabase.DB.QueryRow(`SELECT votes_cast FROM users WHERE tg_user_id=10`).Scan(&v10); err != nil {
		t.Fatalf("select votes_cast u10: %v", err)
	}
	if err := testDatabase.DB.QueryRow(`SELECT votes_cast FROM users WHERE tg_user_id=20`).Scan(&v20); err != nil {
		t.Fatalf("select votes_cast u20: %v", err)
	}

	if v10 != 5 || v20 != 2 {
		t.Fatalf("expected votes_cast u10=5 u20=2, got u10=%d u20=%d", v10, v20)
	}
}
