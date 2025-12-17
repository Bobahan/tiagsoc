package repository

import (
	"context"
	"database/sql"
	"social/internal/storage/postgresql"

	"github.com/lib/pq"
)

type Followers struct {
	db *sql.DB
}

func FollowersRepository(db *sql.DB) *Followers {
	return &Followers{db}
}

func (f *Followers) Follow(ctx context.Context, userId, followerId int64) error {
	query := `INSERT INTO followers (user_id, follower_id) VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	_, err := f.db.ExecContext(ctx, query, userId, followerId)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return postgresql.ErrorDuplicate
		}
	}

	return nil
}

func (f *Followers) Unfollow(ctx context.Context, userId, followerId int64) error {
	query := `DELETE FROM followers WHERE user_id = $1 AND follower_id = $2`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	_, err := f.db.ExecContext(ctx, query, userId, followerId)
	return err
}
