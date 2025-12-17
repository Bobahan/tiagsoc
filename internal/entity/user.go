package entity

type User struct {
	Id        int64      `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Password  string     `json:"-"`
	CreatedAt string     `json:"created_at"`
	Followers *Followers `json:"followers"` // отношение
}

type Users []*User
