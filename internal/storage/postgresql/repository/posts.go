package repository

import (
	"context"
	"database/sql"
	"errors"
	"social/internal/entity"
	"social/internal/service"
	"social/internal/storage/postgresql"

	"github.com/lib/pq"
)

type Posts struct {
	db *sql.DB
}

func PostRepository(db *sql.DB) *Posts {
	return &Posts{db}
}

func (p *Posts) Create(ctx context.Context, post *entity.Post) error {
	query := `INSERT INTO posts (content, title, user_id, tags) VALUES($1, $2, $3, $4) RETURNING id, created_at, updated_at`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := p.db.QueryRowContext(
		ctx,
		query,
		post.Content,
		post.Title,
		post.UserId,
		pq.Array(post.Tags), // post.Tags - это slice, нужно привести в Array
	).Scan(&post.Id, &post.CreatedAt, &post.UpdatedAt); err != nil {
		return err
	}

	return nil
}

func (p *Posts) GetById(ctx context.Context, id int64) (*entity.Post, error) {
	query := `SELECT id, content, title, user_id, tags, created_at, updated_at, version FROM posts WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	var post entity.Post

	if err := p.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&post.Id,
		&post.Content,
		&post.Title,
		&post.UserId,
		pq.Array(&post.Tags),
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Version,
	); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, postgresql.ErrorNotFound
		default:
			return nil, err
		}
	}

	return &post, nil
}

func (p *Posts) Update(ctx context.Context, post *entity.Post) error {
	query := `UPDATE posts SET title = $1, content = $2, version = version + 1 WHERE posts.id = $3 AND version = $4 RETURNING version`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := p.db.QueryRowContext(
		ctx,
		query,
		&post.Title,
		&post.Content,
		&post.Id,
		&post.Version,
	).Scan(&post.Version); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return postgresql.ErrorNotFound
		default:
			return err
		}
	}

	return nil
}

func (p *Posts) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM posts WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	res, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 { // если ничего не затронуто значит мы ничего не удалили
		return postgresql.ErrorNotFound
	}

	return nil
}

func (p *Posts) GetUserFeed(ctx context.Context, userId int64, pagination *service.Pagination) (*entity.Feeds, error) {
	query := `
    SELECT p.id, p.title, p.content, p.user_id, p.version, u.username, p.tags, count(c.id) as comments_count 
    FROM posts p 
    LEFT JOIN "comments" c ON c.post_id = p.id
    LEFT JOIN users u ON p.user_id = u.id 
    JOIN followers f ON p.user_id = f.follower_id OR p.user_id = $1
    WHERE (f.user_id = $1 OR p.user_id = $1) AND (p.title ILIKE $4 OR p.content ILIKE $4)
    GROUP BY p.id, u.username
    ORDER BY p.created_at ` + pagination.Sort + `
    LIMIT $2 OFFSET $3
  `

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	rows, err := p.db.QueryContext(
		ctx,
		query,
		userId,
		pagination.Limit,
		pagination.Offset,
		pagination.Search+"%",
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var feed entity.Feeds

	for rows.Next() {
		var post entity.Feed

		if err := rows.Scan(
			&post.Id,
			&post.Title,
			&post.Content,
			&post.UserId,
			&post.Version,
			&post.User.Username,
			pq.Array(&post.Tags),
			&post.CommentCount,
		); err != nil {
			return nil, err
		}

		feed = append(feed, &post)
	}

	return &feed, nil
}
