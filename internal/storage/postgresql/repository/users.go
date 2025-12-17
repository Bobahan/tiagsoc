package repository

import (
	"context"
	"database/sql"
	"errors"
	"social/internal/entity"
	"social/internal/storage/postgresql"
)

type Users struct {
	db *sql.DB
}

func UserRepository(db *sql.DB) *Users {
	return &Users{db}
}

func (u *Users) Create(ctx context.Context, user *entity.User) error {
	query := `INSERT INTO users (username, password, email) VALUES($1, $2, $3) RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := u.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Password,
		user.Email,
	).Scan(
		&user.Id,
		&user.CreatedAt,
	); err != nil {
		return err
	}

	return nil
}

func (u *Users) GetById(ctx context.Context, id int64) (*entity.User, error) {
	query := `SELECT id, username, email, password, created_at FROM users WHERE id = $1`

	var user entity.User

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := u.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, postgresql.ErrorNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
