package entity

type Feed struct {
	Post
	CommentCount int `json:"comment_count"`
}

type Feeds []*Feed
