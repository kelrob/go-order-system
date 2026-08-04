package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/notification-service/internal/config"
	"github.com/kelrob/notification-service/internal/kafka"
	"github.com/kelrob/notification-service/internal/notification"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
)

var (
	db     *pgxpool.Pool
	appLog *logger.Logger
)

func isProcessed(ctx context.Context, eventId string) (bool, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM processed_events WHERE event_id = $1`,
		eventId,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func markProcessed(ctx context.Context, eventId string) error {
	_, err := db.Exec(ctx,
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		eventId,
	)
	return err
}

func handlePaymentSucceeded(data []byte) error {
	var event notification.PaymentSucceededEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	processed, err := isProcessed(context.Background(), event.OrderId)
	if err != nil {
		return fmt.Errorf("failed to check processed events: %w", err)
	}

	if processed {
		appLog.Warn(event.TraceId, events.PaymentSucceeded, "Event already processed, skipping", nil)
		return nil
	}

	appLog.Info(event.TraceId, events.PaymentSucceeded, "Sending confirmation email", map[string]any{
		"user_id":  event.UserId,
		"order_id": event.OrderId,
	})

	err = markProcessed(context.Background(), event.OrderId)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	return nil
}

func main() {
	var err error
	cfg := config.Load()
	appLog = logger.NewLogger("notification-service")

	db, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	appLog.Log("Connected to database successfully", nil)

	dlq, err := kafka.NewDLQ([]string{"localhost:9092"}, appLog)
	if err != nil {
		log.Fatal("Failed to create DLQ:", err)
	}
	defer dlq.Close()

	consumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.PaymentSucceeded,
		"notification-service",
		handlePaymentSucceeded,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}
	defer consumer.Close()

	appLog.Log("Notification service started", nil)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	mux := http.NewServeMux()
	notification.Register(mux)

	go func() {
		appLog.Log("Listening on port "+cfg.Port, nil)
		if err := http.ListenAndServe(":"+cfg.Port, appLog.Middleware(mux)); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	if err := consumer.Start(ctx); err != nil {
		log.Fatal("Consumer error:", err)
	}
}
