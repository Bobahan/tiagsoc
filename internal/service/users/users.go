package user_service

import (
	"context"
	"social/internal/entity"
	"time"
)

// Похуй какую БД использовать главное чтобы БД удовлетворял interface, те должен быть метод Save
type Repository interface {
	Create(context.Context, *entity.User) error
	GetById(ctx context.Context, id int64) (*entity.User, error)
	CreateAndInvite(ctx context.Context, user *entity.User, token string, exp time.Duration) error // создает user в таблице users и user_invitation
	Activate(ctx context.Context, token string) error
	Delete(ctx context.Context, userId int64) error
}

type UsersService struct {
	userRepository Repository
}

func New(userRepository Repository) *UsersService {
	return &UsersService{
		userRepository: userRepository,
	}
}

func (u *UsersService) GetUser(ctx context.Context, id int64) (*entity.User, error) {
	user, err := u.userRepository.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *UsersService) RegisterUser(ctx context.Context, user *entity.User, token string, exp time.Duration) error {
	if err := u.userRepository.CreateAndInvite(ctx, user, token, exp); err != nil {
		return err
	}

	return nil
}

func (u *UsersService) ActivateUser(ctx context.Context, token string) error {
	if err := u.userRepository.Activate(ctx, token); err != nil {
		return err
	}

	return nil
}

func (u *UsersService) Delete(ctx context.Context, userId int64) error {
	return nil
}
