package user_service

import (
	"context"
	"social/internal/entity"
)

// Похуй какую БД использовать главное чтобы БД удовлетворял interface, те должен быть метод Save
type Repository interface {
	Create(context.Context, *entity.User) error
	GetById(ctx context.Context, id int64) (*entity.User, error)
}

type UsersService struct {
	userRepository Repository
}

func New(userRepository Repository) *UsersService {
	return &UsersService{
		userRepository: userRepository,
	}
}

func (u *UsersService) Get(ctx context.Context, id int64) (*entity.User, error) {
	user, err := u.userRepository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}
