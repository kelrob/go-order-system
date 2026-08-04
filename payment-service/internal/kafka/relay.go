package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/shared/logger"
)

type Relay struct {
	db       *pgxpool.Pool
	producer *Producer
	appLog   *logger.Logger
}

func NewRelay(db *pgxpool.Pool, producer *Producer, appLog *logger.Logger) *Relay {
	return &Relay{db: db, producer: producer, appLog: appLog}
}

func (r *Relay) Start(ctx context.Context) {
	r.appLog.Log("Outbox relay started", nil)

	for {
		select {
		case <-ctx.Done():
			r.appLog.Log("Relay shutting down", nil)
			return
		default:
			err := r.processOutbox(ctx)
			if err != nil {
				r.appLog.Error("", "relay", "Relay error", err, nil)
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *Relay) processOutbox(ctx context.Context) error {
	rows, err := r.db.Query(ctx,
		`SELECT id, event_type, payload 
		FROM outbox 
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 10`,
	)
	if err != nil {
		return fmt.Errorf("failed to query outbox: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, eventType string
		var payload []byte

		err := rows.Scan(&id, &eventType, &payload)
		if err != nil {
			return fmt.Errorf("failed to scan outbox row: %w", err)
		}

		var envelope struct {
			TraceId string `json:"trace_id"`
		}
		json.Unmarshal(payload, &envelope)

		var data any
		err = json.Unmarshal(payload, &data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		err = r.producer.Publish(ctx, eventType, data)
		if err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		_, err = r.db.Exec(ctx,
			`UPDATE outbox SET status = 'published', updated_at = NOW() WHERE id = $1`,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to update outbox status: %w", err)
		}

		r.appLog.Info(envelope.TraceId, eventType, "Published event", map[string]any{
			"id":         id,
			"event_type": eventType,
		})
	}

	return rows.Err()
}
