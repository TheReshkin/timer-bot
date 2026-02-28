package models

type User struct {
	UserID int64   `json:"user_id"`
	ChatID int64   `json:"chat_id"`
	Events []Event `json:"events"`
}
