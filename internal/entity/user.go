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
	RoleId    int64      `json:"role_id"`
	Role      Role       `json:"role"` // отношение
}

type password struct {
	Text *string
	Hash []byte
}

func (p *password) Set(text string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.Text = &text
	p.Hash = hashedPassword

	return nil
}

func (p *password) Get() string {
	return string(p.Hash)
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.Hash, []byte(text))
}

type Users []*User
