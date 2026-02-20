package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kiselevos/memento_game_bot/internal/db"
)

const (
	StatGame  = "game"
	StatVote  = "vote"
	StatPhoto = "photo"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *db.Db) *UserRepo {
	return &UserRepo{
		db: db.DB,
	}
}

func (repo *UserRepo) CreateIfNotExists(ctx context.Context, tgUserID int64, username, firstName string) error {
	_, err := repo.db.ExecContext(ctx, `
INSERT INTO users (tg_user_id, username, first_name, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
ON CONFLICT (tg_user_id) DO NOTHING
`, tgUserID, nullifyEmpty(username), nullifyEmpty(firstName))
	if err != nil {
		return fmt.Errorf("users create if not exists: %w", err)
	}
	return nil
}

func (repo *UserRepo) IncGamesPlayed(ctx context.Context, tgUserID int64) error {
	res, err := repo.db.ExecContext(ctx, `
UPDATE users
SET games_played = games_played + 1,
    updated_at = now()
WHERE tg_user_id = $1
`, tgUserID)
	if err != nil {
		return fmt.Errorf("users inc games_played: %w", err)
	}

	return ensureRowsAffected(res,
		fmt.Sprintf("users inc games_played: user tg_user_id=%d not found", tgUserID))
}

// После раунда инкрементируем голоса и фотографии для юзера
func (repo *UserRepo) IncUsersVotes(ctx context.Context, votesByUser map[int64]int64) error {

	if len(votesByUser) == 0 {
		return nil
	}

	query := `
UPDATE users u
SET
	photos_sent = u.photos_sent + 1,
	votes_cast  = u.votes_cast + v.votes,
	updated_at  = now()
FROM (VALUES `

	args := make([]any, 0, len(votesByUser)*2)
	i := 1
	first := true

	for tgUserID, votes := range votesByUser {
		if !first {
			query += ","
		}
		first = false

		query += fmt.Sprintf("($%d,$%d)", i, i+1)
		args = append(args, tgUserID, votes)
		i += 2
	}

	query += `
) AS v(tg_user_id, votes)
WHERE u.tg_user_id = v.tg_user_id
`
	_, err := repo.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("users batch inc votes: %w", err)
	}

	return nil
}
