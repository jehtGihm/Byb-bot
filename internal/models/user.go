package models

import "time"

// User represents a member in our database.
type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Username   string    `json:"username"`
	JoinedAt   time.Time `json:"joined_at"`
}
