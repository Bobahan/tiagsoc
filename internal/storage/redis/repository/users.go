package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"social/internal/entity"
	"time"

	"github.com/redis/go-redis/v9"
)

const UserExpTime = time.Minute

// маленькая часть Redis Storage
type Users struct {
	redis *redis.Client
}

func UsersRepository(redis *redis.Client) *Users {
	return &Users{redis}
}

func (u *Users) Get(ctx context.Context, userId int64) (*entity.User, error) {
	cacheKey := fmt.Sprintf("user:%v", userId) // user:42

	data, err := u.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil { // cache miss
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// data -- это строка нам нужно это строку расформировать в json структуру

	var user entity.User
	if data != "" {
		if err := json.Unmarshal([]byte(data), &user); err != nil {
			return nil, err
		}
	}

	return &user, nil
}

func (u *Users) Set(ctx context.Context, user *entity.User) error {
	cacheKey := fmt.Sprintf("user:%v", user.Id) // user:42

	json, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return u.redis.SetEx(ctx, cacheKey, json, UserExpTime).Err()
}
