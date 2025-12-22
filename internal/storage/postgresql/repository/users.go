package repository

import (
	"context"
	"database/sql"
	"errors"
	"social/internal/entity"
	"social/internal/storage/postgresql"
	"time"

	"github.com/k0kubun/pp"
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

func (u *Users) CreateAndInvite(ctx context.Context, user *entity.User, token string, invitationExp time.Duration) error {
	// transaction wrapper
	return postgresql.WithTransaction(u.db, ctx, func(tx *sql.Tx) error {

		// create the user
		if err := u.createTx(ctx, tx, user); err != nil {
			return err
		}

		// create the user invite
		if err := u.createInvitationTx(ctx, tx, user.Id, token, invitationExp); err != nil { // invitationExp -- приглашение умайар через какое-то вермя
			return err
		}

		return nil
	})
}

func (u *Users) Activate(ctx context.Context, token string) error {
	return postgresql.WithTransaction(u.db, ctx, func(tx *sql.Tx) error {

		// находим по uuid пользователя
		user, err := u.getUserFromInvitation(ctx, tx, token)
		if err != nil {
			return err
		}

		// обновляем is_active в users
		user.IsActive = true
		if err := u.updateIsActive(ctx, tx, user); err != nil {
			return err
		}

		// удаляем запись если нашли пользователя по uuid в user_invitations
		if err := u.deleteInvitation(ctx, tx, user.Id); err != nil {
			return err
		}

		return nil
	})
}

func (u *Users) Delete(ctx context.Context, userId int64) error {
	return postgresql.WithTransaction(u.db, ctx, func(tx *sql.Tx) error {
		if err := u.delete(ctx, tx, userId); err != nil {
			return err
		}

		if err := u.deleteInvitation(ctx, tx, userId); err != nil {
			return err
		}

		return nil
	})
}

func (u *Users) delete(ctx context.Context, tx *sql.Tx, userId int64) error {
	query := `DELETE FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if _, err := tx.ExecContext(ctx, query, userId); err != nil {
		return err
	}

	return nil
}

func (u *Users) createTx(ctx context.Context, tx *sql.Tx, user *entity.User) error {
	query := `INSERT INTO users (username, password, email) VALUES($1, $2, $3) RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := tx.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Password.Get(),
		user.Email,
	).Scan(
		&user.Id,
		&user.CreatedAt,
	); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return postgresql.ErrorDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return postgresql.ErrorDuplicateUsername
		default:
			return err
		}
	}

	return nil
}

// table user_invitations
func (u *Users) createInvitationTx(
	ctx context.Context,
	tx *sql.Tx,
	userId int64,
	token string,
	invitationExp time.Duration,
) error {
	query := `INSERT INTO user_invitations (user_id, token, expiry) VALUES ($1, $2, $3)`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if _, err := tx.ExecContext(ctx, query, userId, token, time.Now().Add(invitationExp)); err != nil {
		return err
	}

	return nil
}

func (u *Users) getUserFromInvitation(ctx context.Context, tx *sql.Tx, token string) (*entity.User, error) {
	query := `
    SELECT u.id, u.username, u.email, u.created_at, is_active FROM users u
    JOIN user_invitations ui ON u.id = ui.user_id
    WHERE ui.token = $1
  `

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	user := &entity.User{}

	if err := tx.QueryRowContext(
		ctx,
		query,
		token,
	).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.IsActive,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			pp.Println(user)
			return nil, postgresql.ErrorNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (u *Users) updateIsActive(ctx context.Context, tx *sql.Tx, user *entity.User) error {
	query := `UPDATE users SET username = $1, email = $2, is_active = $3 WHERE id = $4`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if _, err := tx.ExecContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.IsActive,
		user.Id,
	); err != nil {
		return err
	}

	return nil
}

func (u *Users) deleteInvitation(ctx context.Context, tx *sql.Tx, userId int64) error {
	query := `DELETE FROM user_invitations WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if _, err := tx.ExecContext(ctx, query, userId); err != nil {
		return err
	}

	return nil
}
