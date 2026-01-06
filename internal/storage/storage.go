package storage

import (
	"database/sql"
	"social/internal/storage/postgresql/repository"
)

type Storage struct {
	Posts     repository.Posts
	Users     repository.Users
	Comments  repository.Comments
	Followers repository.Followers
	Roles     repository.Roles
}

func New(db *sql.DB) *Storage {
	return &Storage{
		Posts:     *repository.PostRepository(db),
		Users:     *repository.UserRepository(db),
		Comments:  *repository.CommentRepository(db),
		Followers: *repository.FollowersRepository(db),
		Roles:     *repository.RolesRepository(db),
	}
}
