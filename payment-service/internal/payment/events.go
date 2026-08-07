package payment

import "time"

// Incoming events
type InventoryReservedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

type OrderItemEvent struct {
	ProductId string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// Outgoing events

type PaymentFailedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

type PaymentSucceededEvent struct {
	EventId   string    `json:"event_id"`
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	UserId    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
