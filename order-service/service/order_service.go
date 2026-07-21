package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kelrob/order-service/domain"
	"github.com/kelrob/order-service/storage"
)

type OrderService struct {
	store *storage.OrderStore
}

func NewOrderService(store *storage.OrderStore) *OrderService {
	return &OrderService{store: store}
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

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (domain.Order, error) {
	if input.UserId == "" {
		return domain.Order{}, fmt.Errorf("user_id is required")
	}

	if len(input.Items) == 0 {
		return domain.Order{}, fmt.Errorf("order must have at least one item")
	}

	order := domain.Order{
		Id:        fmt.Sprintf("%d", time.Now().UnixNano()),
		UserId:    input.UserId,
		TraceId:   input.TraceId,
		Status:    domain.OrderStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, item := range input.Items {
		order.Items = append(order.Items, domain.OrderItem{
			OrderId:   order.Id,
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Status:    domain.OrderStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	err := s.store.CreateOrder(ctx, order)
	if err != nil {
		return domain.Order{}, fmt.Errorf("failed to save order: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrders(ctx context.Context) ([]domain.Order, error) {
	orders, err := s.store.GetOrders(ctx)
	if err != nil {
		return []domain.Order{}, fmt.Errorf("failed to ger orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	order, err := s.store.GetOrder(ctx, id)
	if err != nil {
		return domain.Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}
