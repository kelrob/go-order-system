package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/payment-service/circuit"
	"github.com/kelrob/payment-service/domain"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
)

type PaymentStore struct {
	db      *pgxpool.Pool
	breaker *circuit.Breaker
	appLog  *logger.Logger
}

func NewPaymentStore(db *pgxpool.Pool, appLog *logger.Logger) *PaymentStore {
	return &PaymentStore{db: db, breaker: circuit.NewBreaker(3, 10*time.Second, appLog), appLog: appLog}
}

func (s *PaymentStore) ProcessPayment(ctx context.Context, event domain.InventoryReservedEvent) error {
	return s.breaker.Execute(func() error {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Simulate payment — 30% chance of failure
		paymentSucceeded := rand.Intn(10) > 2

		paymentStatus := "succeeded"
		eventType := events.PaymentSucceeded
		var outboxPayload []byte

		if paymentSucceeded {
			outboxPayload, err = json.Marshal(domain.PaymentSucceededEvent{
				TraceId:   event.TraceId,
				OrderId:   event.OrderId,
				UserId:    event.UserId,
				CreatedAt: time.Now(),
			})
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to marshal payment succeeded event: %w", err)
			}
		} else {
			paymentStatus = "failed"
			eventType = events.PaymentFailed
			outboxPayload, err = json.Marshal(domain.PaymentFailedEvent{
				TraceId:   event.TraceId,
				OrderId:   event.OrderId,
				UserId:    event.UserId,
				CreatedAt: time.Now(),
				Items:     event.Items,
			})
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to marshal payment failed event: %w", err)
			}
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO payments (id, order_id, user_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())`,
			fmt.Sprintf("%d", time.Now().UnixNano()),
			event.OrderId,
			event.UserId,
			paymentStatus,
		)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to insert payment: %w", err)
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
			eventType,
			outboxPayload,
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

		s.appLog.Info(event.TraceId, eventType, "Payment for order", map[string]any{
			"payment_status": paymentStatus,
			"order_id":       event.OrderId,
		})

		if !paymentSucceeded {
			return fmt.Errorf("payment failed for order %s", event.OrderId)
		}

		return nil
	})
}
