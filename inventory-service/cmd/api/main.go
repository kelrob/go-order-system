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
	"github.com/kelrob/inventory-service/internal/config"
	"github.com/kelrob/inventory-service/internal/inventory"
	"github.com/kelrob/inventory-service/internal/kafka"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
)

var (
	db     *pgxpool.Pool
	repo   *inventory.Repository
	appLog *logger.Logger
)

func handleOrderCreated(data []byte) error {
	var event inventory.OrderCreatedEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	appLog.Info(event.TraceId, events.OrderCreated, "Reserving inventory", map[string]any{
		"order_id": event.OrderId,
	})

	err = repo.ReserveInventory(context.Background(), event)
	if err != nil {
		return fmt.Errorf("failed to reserve inventory: %w", err)
	}

	appLog.Info(event.TraceId, events.OrderCreated, "Inventory reserved", map[string]any{
		"order_id": event.OrderId,
	})
	return nil
}

func handlePaymentFailed(data []byte) error {
	var event inventory.PaymentFailedEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	appLog.Info(event.TraceId, events.PaymentFailed, "Unreserving inventory for order", map[string]any{
		"event_id": event.OrderId,
		"topic":    events.PaymentFailed,
	})

	err = repo.UnreserveInventory(context.Background(), event.OrderId, event.TraceId, event.Items)
	if err != nil {
		return fmt.Errorf("failed to unreserve inventory: %w", err)
	}

	appLog.Info(event.TraceId, events.PaymentFailed, "Inventory unreserved for order", map[string]any{
		"event_id": event.OrderId,
		"topic":    events.PaymentFailed,
	})
	return nil
}

func main() {
	var err error
	cfg := config.Load()
	appLog = logger.NewLogger("inventory-service")
	db, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	appLog.Log("Connected to inventory database successfully", nil)

	repo = inventory.NewRepository(db)

	producer, err := kafka.NewProducer([]string{cfg.KafkaBroker})
	if err != nil {
		log.Fatal("Failed to create producer:", err)
	}
	defer producer.Close()

	relay := kafka.NewRelay(db, producer, appLog)

	dlq, err := kafka.NewDLQ([]string{cfg.KafkaBroker}, appLog)
	if err != nil {
		log.Fatal("Failed to create DLQ:", err)
	}
	defer dlq.Close()

	// order created consumer
	orderConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.OrderCreated,
		"inventory-service",
		handleOrderCreated,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create order consumer:", err)
	}
	defer orderConsumer.Close()

	// payment failed consumer
	paymentConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.PaymentFailed,
		"inventory-service-payment",
		handlePaymentFailed,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create payment consumer:", err)
	}
	defer paymentConsumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go relay.Start(ctx)
	go orderConsumer.Start(ctx)

	appLog.Log("Inventory service started", nil)

	mux := http.NewServeMux()
	inventory.Register(mux)

	go func() {
		appLog.Log("Listening on port "+cfg.Port, nil)
		if err := http.ListenAndServe(":"+cfg.Port, appLog.Middleware(mux)); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	if err := paymentConsumer.Start(ctx); err != nil {
		log.Fatal("Payment consumer error:", err)
	}
}
