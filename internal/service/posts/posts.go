package post_service

import (
	"context"
	"social/internal/entity"
)

type PostRepository interface {
	Create(context.Context, *entity.Post) error
	GetById(ctx context.Context, id int64) (*entity.Post, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post *entity.Post) error
}

type PostsService struct {
	postRepository PostRepository
}

func New(postRepository PostRepository) *PostsService {
	return &PostsService{
		postRepository: postRepository,
	}
}

func (p *PostsService) Create(ctx context.Context, post *entity.Post) error {
	if err := p.postRepository.Create(ctx, post); err != nil {
		return err
	}

	return nil
}

func (p *PostsService) Get(ctx context.Context, id int64) (*entity.Post, error) {
	post, err := p.postRepository.GetById(ctx, id)

	if err != nil {
		return nil, err
	}

	return post, nil
}

func (p *PostsService) Delete(ctx context.Context, id int64) error {
	if err := p.postRepository.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (p *PostsService) Update(ctx context.Context, post *entity.Post) error {
	if err := p.postRepository.Update(ctx, post); err != nil {
		return err
	}

	return nil
}
