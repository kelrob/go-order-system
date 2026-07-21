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
	"github.com/kelrob/payment-service/config"
	"github.com/kelrob/payment-service/domain"
	"github.com/kelrob/payment-service/kafka"
	"github.com/kelrob/payment-service/storage"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
)

var (
	db           *pgxpool.Pool
	paymentStore *storage.PaymentStore
	appLog       *logger.Logger
)

func handleInventoryReserved(data []byte) error {
	var event domain.InventoryReservedEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	appLog.Info(event.TraceId, events.InventoryReserved, "Processing payment for order", map[string]any{
		"order_id": event.OrderId,
	})

	err = paymentStore.ProcessPayment(context.Background(), event)
	if err != nil {
		return fmt.Errorf("failed to process payment: %w", err)
	}

	return nil
}

func main() {
	var err error
	cfg := config.Load()
	appLog = logger.NewLogger("payment-service")
	db, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	appLog.Log("Connected to payment database successfully", nil)

	paymentStore = storage.NewPaymentStore(db, appLog)

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

	consumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.InventoryReserved,
		"payment-service",
		handleInventoryReserved,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}
	defer consumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go relay.Start(ctx)

	appLog.Log("Payment service started", nil)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "payment-service",
		})
	})

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
