package auth

import "time"

type Role string

const (
	AdminRole Role = "admin"
	UserRole  Role = "user"
)

type User struct {
	Id             string    `json:"id"`
	TraceId        string    `json:"trace_id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	HashedPassword string    `json:"-"`
	Role           Role      `json:"role"`
	IsLocked       bool      `json:"is_locked"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
