package order

import "time"

type OrderItemEvent struct {
	ProductId string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderCreatedEvent struct {
	TraceId   string           `json:"trace_id"`
	OrderId   string           `json:"order_id"`
	UserId    string           `json:"user_id"`
	Status    OrderStatus      `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	Items     []OrderItemEvent `json:"items"`
}

type InventoryUnreservedEvent struct {
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PaymentSucceededEvent struct {
	TraceId   string    `json:"trace_id"`
	OrderId   string    `json:"order_id"`
	UserId    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
