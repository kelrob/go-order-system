package order

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateOrderInput struct {
	UserId  string
	TraceId string
	Items   []OrderItemInput
}

type OrderItemInput struct {
	ProductId string
	Quantity  int
	Price     float64
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (Order, error) {
	if input.UserId == "" {
		return Order{}, fmt.Errorf("user_id is required")
	}

	if len(input.Items) == 0 {
		return Order{}, fmt.Errorf("order must have at least one item")
	}

	order := Order{
		Id:        fmt.Sprintf("%d", time.Now().UnixNano()),
		UserId:    input.UserId,
		TraceId:   input.TraceId,
		Status:    OrderStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, item := range input.Items {
		order.Items = append(order.Items, OrderItem{
			OrderId:   order.Id,
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Status:    OrderStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		return Order{}, fmt.Errorf("failed to save order: %w", err)
	}

	return order, nil
}

func (s *Service) GetOrders(ctx context.Context) ([]Order, error) {
	orders, err := s.repo.GetOrders(ctx)
	if err != nil {
		return []Order{}, fmt.Errorf("failed to ger orders: %w", err)
	}

	return orders, nil
}

func (s *Service) GetOrder(ctx context.Context, id string) (Order, error) {
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}
