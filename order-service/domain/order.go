package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusFailed    OrderStatus = "failed"
)

type Order struct {
	Id        string      `json:"id"`
	TraceId   string      `json:"trace_id"`
	UserId    string      `json:"user_id"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Items     []OrderItem `json:"items"`
}

type OrderItem struct {
	OrderId   string      `json:"order_id"`
	ProductId string      `json:"product_id"`
	Quantity  int         `json:"quantity"`
	Price     float64     `json:"price"`
	Status    OrderStatus `json:"status"` // Product could be delivered separately, so we need to track the status of each product in the order
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func (o Order) TotalPrice() float64 {
	var total float64
	for _, item := range o.Items {
		total += item.Price * float64(item.Quantity)
	}

	return total
}

func (o *Order) Confirm() {
	o.Status = OrderStatusConfirmed
	o.UpdatedAt = time.Now()
}
