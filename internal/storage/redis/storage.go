package cache

import (
	"context"
	"social/internal/entity"
	"social/internal/storage/redis/repository"

	"github.com/redis/go-redis/v9"
)

type Users interface {
	Get(context.Context, int64) (*entity.User, error)
	Set(context.Context, *entity.User) error
}

// Весь Storage of Redis
type Storage struct {
	Users
}

// Инициализирует Storagage
func New(redis *redis.Client) Storage {
	return Storage{
		Users: repository.UsersRepository(redis), // это определенный кусок storage: Users
	}
}
