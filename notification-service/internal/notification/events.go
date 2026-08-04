package notification

import "time"

type PaymentSucceededEvent struct {
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	UserId    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
