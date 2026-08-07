package notification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/ulid"
)

type Repository struct {
	db     *pgxpool.Pool
	appLog *logger.Logger
}

func NewRepository(db *pgxpool.Pool, appLog *logger.Logger) *Repository {
	return &Repository{db: db, appLog: appLog}
}

func (r *Repository) isProcessed(ctx context.Context, eventId string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM processed_events WHERE event_id = $1`,
		eventId,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// recordNotification marks the event processed and logs the notification atomically,
// so a crash between the two never leaves us with a sent notification we'd resend.
func (r *Repository) recordNotification(ctx context.Context, eventId, userId, notifType, recipient string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO notifications (id, user_id, type, channel, recipient, status, created_at)
		 VALUES ($1, $2, $3, 'email', $4, 'sent', NOW())`,
		ulid.Generate(), userId, notifType, recipient,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert notification: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		eventId,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert processed event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) SendPaymentConfirmation(ctx context.Context, event PaymentSucceededEvent) error {
	processed, err := r.isProcessed(ctx, event.EventId)
	if err != nil {
		return fmt.Errorf("failed to check processed events: %w", err)
	}
	if processed {
		r.appLog.Warn(event.TraceId, events.PaymentSucceeded, "Event already processed, skipping", nil)
		return nil
	}

	if err := r.recordNotification(ctx, event.EventId, event.UserId, "payment_succeeded", event.UserId); err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}

	r.appLog.Info(event.TraceId, events.PaymentSucceeded, "Sending confirmation email", map[string]any{
		"user_id":  event.UserId,
		"order_id": event.OrderId,
	})

	return nil
}

func (r *Repository) SendWelcomeEmail(ctx context.Context, event UserRegisteredEvent) error {
	processed, err := r.isProcessed(ctx, event.EventId)
	if err != nil {
		return fmt.Errorf("failed to check processed events: %w", err)
	}
	if processed {
		r.appLog.Warn(event.TraceId, events.UserRegistered, "Event already processed, skipping", nil)
		return nil
	}

	if err := r.recordNotification(ctx, event.EventId, event.UserId, "user_registered", event.Email); err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}

	r.appLog.Info(event.TraceId, events.UserRegistered, "Sending welcome email", map[string]any{
		"user_id": event.UserId,
		"email":   event.Email,
	})

	return nil
}
