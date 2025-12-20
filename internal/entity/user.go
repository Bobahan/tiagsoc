package entity

import "golang.org/x/crypto/bcrypt"

type User struct {
	Id        int64      `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Password  password   `json:"-"`
	CreatedAt string     `json:"created_at"`
	IsActive  bool       `json:"is_active"`
	Followers *Followers `json:"followers"` // отношение
}

type password struct {
	text *string
	hash []byte
}

func (p *password) Set(text string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.text = &text
	p.hash = hashedPassword

	return nil
}

func (p *password) Get() string {
	return string(p.hash)
}

type Users []*User
