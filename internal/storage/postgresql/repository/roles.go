package repository

import (
	"context"
	"database/sql"
	"social/internal/entity"
	"social/internal/storage/postgresql"
)

type Roles struct {
	db *sql.DB
}

func RolesRepository(db *sql.DB) *Roles {
	return &Roles{db}
}

func (r *Roles) GetByName(ctx context.Context, roleName string) (*entity.Role, error) {
	query := `SELECT name, level, description FROM roles WHERE roles.name = $1`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	role := &entity.Role{}

	if err := r.db.QueryRowContext(
		ctx,
		query,
		roleName,
	).Scan(
		&role.Name,
		&role.Level,
		&role.Description,
	); err != nil {
		return nil, err
	}

	return role, nil
}
