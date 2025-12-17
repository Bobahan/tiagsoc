package followers_service

import (
	"context"
)

type FollowersRepository interface {
	Follow(ctx context.Context, userId, followerId int64) error
	Unfollow(ctx context.Context, userId, followerId int64) error
}

type FollowersService struct {
	followersRepository FollowersRepository
}

func New(followersRepository FollowersRepository) *FollowersService {
	return &FollowersService{
		followersRepository: followersRepository,
	}
}

func (f *FollowersService) FollowUser(ctx context.Context, userId, followerId int64) error {
	if err := f.followersRepository.Follow(ctx, userId, followerId); err != nil {
		return err
	}

	return nil
}

func (f *FollowersService) UnfollowUser(ctx context.Context, userId, followerId int64) error {
	if err := f.followersRepository.Unfollow(ctx, userId, followerId); err != nil {
		return err
	}

	return nil
}

// Unfollow(ctx context.Context, userId, followerId int64) error
