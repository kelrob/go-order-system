package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/inventory-service/domain"
	"github.com/kelrob/shared/events"
)

type InventoryStore struct {
	db *pgxpool.Pool
}

func NewInventoryStore(db *pgxpool.Pool) *InventoryStore {
	return &InventoryStore{db: db}
}

func (s *InventoryStore) ReserveInventory(ctx context.Context, event domain.OrderCreatedEvent) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, item := range event.Items {
		result, err := tx.Exec(ctx,
			`UPDATE inventory_items 
			SET reserved = reserved + $1, available = available - $1
			WHERE product_id = $2 AND available >= $1`,
			item.Quantity,
			item.ProductId,
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to reserve inventory for product %s: %w", item.ProductId, err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			tx.Rollback(ctx)
			return fmt.Errorf("insufficient inventory for product %s", item.ProductId)
		}
	}

	payload, err := json.Marshal(domain.InventoryReservedEvent{
		TraceId:   event.TraceId,
		OrderId:   event.OrderId,
		UserId:    event.UserId,
		CreatedAt: time.Now(),
		Items:     event.Items,
	})
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		event.OrderId,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert processed event: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		events.InventoryReserved,
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

func (s *InventoryStore) UnreserveInventory(ctx context.Context, orderId string, traceId string, items []domain.OrderItemEvent) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`UPDATE inventory_items 
			SET reserved = reserved - $1, available = available + $1
			WHERE product_id = $2`,
			item.Quantity,
			item.ProductId,
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to unreserve inventory for product %s: %w", item.ProductId, err)
		}
	}

	payload, err := json.Marshal(domain.InventoryUnreservedEvent{
		TraceId:   traceId,
		OrderId:   orderId,
		CreatedAt: time.Now(),
	})
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		events.InventoryUnreserved,
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
