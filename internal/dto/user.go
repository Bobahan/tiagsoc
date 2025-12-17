package dto

type FollowUserRequest struct {
	UserId int64 `json:"user_id" validate:"required"`
}
