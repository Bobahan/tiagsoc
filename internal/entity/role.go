package entity

type Role struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Level       int64  `json:"level"`
	Description string `json:"description"`
}
