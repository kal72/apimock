package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
