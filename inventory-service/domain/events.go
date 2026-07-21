package domain

import "time"

// Incoming events
type OrderCreatedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	Status    string           `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

type OrderItemEvent struct {
	ProductId string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type PaymentFailedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

// Outgoing events
type InventoryReservedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

type InventoryUnreservedEvent struct {
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}
