package domain

import "time"

type UserRegisteredEvent struct {
	TraceId   string    `json:"trace_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}
