package repository

import (
	"context"
	"database/sql"
	"social/internal/entity"
	"social/internal/storage/postgresql"
)

type Comments struct {
	db *sql.DB
}

func CommentRepository(db *sql.DB) *Comments {
	return &Comments{db}
}

func (p *Comments) GetByPostId(ctx context.Context, postId int64) (*entity.Comments, error) {
	query := `
    SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, users.username, users.id FROM comments c
    JOIN users ON c.user_id = users.id
    WHERE c.post_id = $1
    ORDER BY c.created_at DESC
  `

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	rows, err := p.db.QueryContext(ctx, query, postId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	comments := entity.Comments{}

	for rows.Next() {
		var comment entity.Comment
		comment.User = entity.User{}

		if err := rows.Scan(
			&comment.Id,
			&comment.PostId,
			&comment.UserId,
			&comment.Content,
			&comment.CreatedAt,
			&comment.User.Username,
			&comment.User.Id,
		); err != nil {
			return nil, err
		}

		comments = append(comments, &comment)
	}

	return &comments, nil
}

func (p *Comments) Create(ctx context.Context, comment *entity.Comment) error {
	query := `INSERT INTO comments (post_id, user_id, content) VALUES($1, $2, $3) RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(ctx, postgresql.QueryTimeoutDuration)
	defer cancel()

	if err := p.db.QueryRowContext(
		ctx,
		query,
		comment.PostId,
		comment.UserId,
		comment.Content,
	).Scan(&comment.Id, &comment.CreatedAt); err != nil {
		return err
	}

	return nil
}
