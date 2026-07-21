package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/order-service/domain"
	"github.com/kelrob/shared/events"
)

type OrderStore struct {
	db *pgxpool.Pool
}

func NewOrderStore(db *pgxpool.Pool) *OrderStore {
	return &OrderStore{db: db}
}

func (s *OrderStore) CreateOrder(ctx context.Context, order domain.Order) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transactions: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO orders (id, user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`,
		order.Id,
		order.UserId,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert order %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (id, order_id, product_id, quantity, price, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			fmt.Sprintf("%d", time.Now().UnixNano()),
			item.OrderId,
			item.ProductId,
			item.Quantity,
			item.Price,
			item.Status,
			item.CreatedAt,
			item.UpdatedAt,
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to insert order items %w", err)
		}
	}

	payload, err := json.Marshal(domain.OrderCreatedEvent{
		TraceId:   order.TraceId,
		OrderId:   order.Id,
		UserId:    order.UserId,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
		Items: func() []domain.OrderItemEvent {
			var items []domain.OrderItemEvent
			for _, item := range order.Items {
				items = append(items, domain.OrderItemEvent{
					ProductId: item.ProductId,
					Quantity:  item.Quantity,
					Price:     item.Price,
				})
			}
			return items
		}(),
	})
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to marshal outbox payload %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		events.OrderCreated,
		payload,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert outbox entry: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *OrderStore) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	var order domain.Order

	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, status, created_at, updated_at
		FROM orders WHERE id = $1`,
		strings.TrimSpace(id),
	).Scan(
		&order.Id,
		&order.UserId,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return domain.Order{}, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT order_id, product_id, quantity, price, status, created_at, updated_at
		FROM order_items WHERE order_id = $1`,
		id,
	)
	if err != nil {
		return domain.Order{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		err := rows.Scan(
			&item.OrderId,
			&item.ProductId,
			&item.Quantity,
			&item.Price,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return domain.Order{}, err
		}
		order.Items = append(order.Items, item)
	}

	if err = rows.Err(); err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

func (s *OrderStore) GetOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.db.Query(ctx,
		`SELECT 
			o.id, o.user_id, o.status, o.created_at, o.updated_at,
			oi.order_id, oi.product_id, oi.quantity, oi.price, oi.status, oi.created_at, oi.updated_at
		FROM orders o
		LEFT JOIN order_items oi ON oi.order_id = o.id
		ORDER BY o.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orderMap := make(map[string]*domain.Order)
	var orderIds []string

	for rows.Next() {
		var o domain.Order
		var item domain.OrderItem

		var itemOrderId *string
		var itemProductId *string
		var itemQuantity *int
		var itemPrice *float64
		var itemStatus *domain.OrderStatus
		var itemCreatedAt *time.Time
		var itemUpdatedAt *time.Time

		err := rows.Scan(
			&o.Id, &o.UserId, &o.Status, &o.CreatedAt, &o.UpdatedAt,
			&itemOrderId, &itemProductId, &itemQuantity, &itemPrice, &itemStatus, &itemCreatedAt, &itemUpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if _, exists := orderMap[o.Id]; !exists {
			orderMap[o.Id] = &o
			orderIds = append(orderIds, o.Id)
		}

		if itemOrderId != nil {
			item = domain.OrderItem{
				OrderId:   *itemOrderId,
				ProductId: *itemProductId,
				Quantity:  *itemQuantity,
				Price:     *itemPrice,
				Status:    *itemStatus,
				CreatedAt: *itemCreatedAt,
				UpdatedAt: *itemUpdatedAt,
			}
			orderMap[o.Id].Items = append(orderMap[o.Id].Items, item)
		}
	}

	var orders []domain.Order
	for _, id := range orderIds {
		orders = append(orders, *orderMap[id])
	}

	return orders, nil
}

func (s *OrderStore) UpdateOrderStatus(ctx context.Context, orderId string, status domain.OrderStatus) error {
	_, err := s.db.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		status,
		orderId,
	)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}
