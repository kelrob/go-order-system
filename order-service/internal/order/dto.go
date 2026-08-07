package order

type CreateOrderRequest struct {
	UserId string             `json:"user_id" validate:"required"`
	Items  []OrderItemRequest `json:"items" validate:"required"`
}

type OrderItemRequest struct {
	ProductId string  `json:"product_id" validate:"required"`
	Quantity  int     `json:"quantity" validate:"required"`
	Price     float64 `json:"price" valudate:"required"`
}
