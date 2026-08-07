package notification

import "time"

// Incoming
type PaymentSucceededEvent struct {
	EventId   string    `json:"event_id"`
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	UserId    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type UserRegisteredEvent struct {
	EventId   string    `json:"event_id"`
	TraceId   string    `json:"trace_id"`
	UserId    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}
