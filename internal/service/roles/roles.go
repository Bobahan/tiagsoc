package roles_service

import (
	"context"
	"social/internal/entity"
)

type RolesRepository interface {
	GetByName(ctx context.Context, roleName string) (*entity.Role, error)
}

type FollowersService struct {
	rolesRepository RolesRepository
}

func New(rolesRepository RolesRepository) *FollowersService {
	return &FollowersService{
		rolesRepository: rolesRepository,
	}
}

func (f *FollowersService) GetByName(ctx context.Context, roleName string) (*entity.Role, error) {
	role, err := f.rolesRepository.GetByName(ctx, roleName)
	if err != nil {
		return nil, err
	}

	return role, nil
}
