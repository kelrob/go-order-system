package auth

import "time"

type UserRegisteredEvent struct {
	EventId   string    `json:"event_id"`
	TraceId   string    `json:"trace_id"`
	UserId    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}
