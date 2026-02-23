package repositories

import (
	"context"
	"testing"
)

func insertTask(t *testing.T, text string) int64 {
	t.Helper()

	var id int64
	err := testDatabase.DB.QueryRow(`
		INSERT INTO tasks (text, is_active)
		VALUES ($1, TRUE)
		RETURNING id
	`, text).Scan(&id)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	return id
}

func TestTaskRepo_IncUse(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewTaskRepo(testDatabase)

	id := insertTask(t, "test task inc use")

	var use0, skip0, photo0 int64
	if err := testDatabase.DB.QueryRow(`SELECT use_count, skip_count, photo_count FROM tasks WHERE id=$1`, id).
		Scan(&use0, &skip0, &photo0); err != nil {
		t.Fatalf("select initial counters: %v", err)
	}

	// use_count +1, photo_count +3
	if err := repo.IncUse(ctx, id, 3); err != nil {
		t.Fatalf("IncUse: %v", err)
	}

	var use1, skip1, photo1 int64
	if err := testDatabase.DB.QueryRow(`SELECT use_count, skip_count, photo_count FROM tasks WHERE id=$1`, id).
		Scan(&use1, &skip1, &photo1); err != nil {
		t.Fatalf("select counters after IncUse: %v", err)
	}

	if use1 != use0+1 {
		t.Fatalf("expected use_count %d, got %d", use0+1, use1)
	}
	if skip1 != skip0 {
		t.Fatalf("expected skip_count unchanged %d, got %d", skip0, skip1)
	}
	if photo1 != photo0+3 {
		t.Fatalf("expected photo_count %d, got %d", photo0+3, photo1)
	}
}

func TestTaskRepo_IncSkip(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewTaskRepo(testDatabase)

	id := insertTask(t, "test task inc skip")

	var use0, skip0, photo0 int64
	if err := testDatabase.DB.QueryRow(`SELECT use_count, skip_count, photo_count FROM tasks WHERE id=$1`, id).
		Scan(&use0, &skip0, &photo0); err != nil {
		t.Fatalf("select initial counters: %v", err)
	}

	if err := repo.IncSkip(ctx, id); err != nil {
		t.Fatalf("IncSkip: %v", err)
	}

	var use1, skip1, photo1 int64
	if err := testDatabase.DB.QueryRow(`SELECT use_count, skip_count, photo_count FROM tasks WHERE id=$1`, id).
		Scan(&use1, &skip1, &photo1); err != nil {
		t.Fatalf("select counters after IncSkip: %v", err)
	}

	if use1 != use0 {
		t.Fatalf("expected use_count unchanged %d, got %d", use0, use1)
	}
	if skip1 != skip0+1 {
		t.Fatalf("expected skip_count %d, got %d", skip0+1, skip1)
	}
	if photo1 != photo0 {
		t.Fatalf("expected photo_count unchanged %d, got %d", photo0, photo1)
	}
}

func TestTaskRepo_IncUse_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewTaskRepo(testDatabase)

	if err := repo.IncUse(ctx, 9999999, 1); err == nil {
		t.Fatalf("expected error for not found task id")
	}
}

func TestTaskRepo_IncSkip_NotFound(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	repo := NewTaskRepo(testDatabase)

	if err := repo.IncSkip(ctx, 9999999); err == nil {
		t.Fatalf("expected error for not found task id")
	}
}
