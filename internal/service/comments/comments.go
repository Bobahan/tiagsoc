package comments_service

import (
	"context"
	"social/internal/entity"
)

type CommentRepository interface {
	GetByPostId(ctx context.Context, postId int64) (*entity.Comments, error)
	Create(ctx context.Context, comment *entity.Comment) error
}

type CommentsService struct {
	commentRepository CommentRepository
}

func New(commentRepository CommentRepository) *CommentsService {
	return &CommentsService{commentRepository}
}

func (c *CommentsService) GetPostComments(ctx context.Context, id int64) (*entity.Comments, error) {
	comments, err := c.commentRepository.GetByPostId(ctx, id)
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (c *CommentsService) CreatePostComment(ctx context.Context, comment *entity.Comment) error {
	if err := c.commentRepository.Create(ctx, comment); err != nil {
		return err
	}

	return nil
}
