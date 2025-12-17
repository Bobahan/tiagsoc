package entity

type Post struct {
	Id        int64     `json:"id"`
	Content   string    `json:"content"`
	Title     string    `json:"title"`
	UserId    int64     `json:"user_id"` // отношение
	Tags      []string  `json:"tags"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	Comments  *Comments `json:"comments"` // отношение
	Version   int       `json:"version"`
	User      User      `json:"user"`
}

type Posts []*Post
