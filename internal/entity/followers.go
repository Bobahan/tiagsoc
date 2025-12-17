package entity

type Follower struct {
	UserId     int64 `json:"user_id"`
	FollowerId int64 `json:"follower_id"`
}

type Followers []*Follower
