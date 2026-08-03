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
	"github.com/kelrob/order-service/config"
	"github.com/kelrob/order-service/domain"
	"github.com/kelrob/order-service/handlers"
	"github.com/kelrob/order-service/kafka"
	"github.com/kelrob/order-service/service"
	"github.com/kelrob/order-service/storage"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/middleware"
)

var (
	db         *pgxpool.Pool
	orderStore *storage.OrderStore
	appLog     *logger.Logger
)

func handlePaymentSucceeded(data []byte) error {
	var event domain.PaymentSucceededEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	err = orderStore.UpdateOrderStatus(context.Background(), event.OrderId, domain.OrderStatusConfirmed)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	appLog.Info(event.TraceId, events.PaymentSucceeded, "Order confirmed", map[string]any{
		"order_id": event.OrderId,
	})
	return nil
}

func handleInventoryUnreserved(data []byte) error {
	var event domain.InventoryUnreservedEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	err = orderStore.UpdateOrderStatus(context.Background(), event.OrderId, domain.OrderStatusFailed)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	appLog.Info(event.TraceId, events.InventoryUnreserved, "Order failed", map[string]any{
		"order_id": event.OrderId,
	})
	return nil
}

func main() {
	var err error
	appLog = logger.NewLogger("order-service")
	cfg := config.Load()

	db, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Unable to connect to database", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		log.Fatal("Unable to ping database", err)
	}

	appLog.Log("Connected to database successfully", nil)

	producer, err := kafka.NewProducer([]string{cfg.KafkaBroker})
	if err != nil {
		log.Fatal("Unable to create kafka producer:", err)
	}
	defer producer.Close()

	orderStore = storage.NewOrderStore(db)
	orderService := service.NewOrderService(orderStore)
	orderHandler := handlers.NewOrderHandler(orderService)

	relay := kafka.NewRelay(db, producer, appLog)

	dlq, err := kafka.NewDLQ([]string{cfg.KafkaBroker}, appLog)
	if err != nil {
		log.Fatal("Unable to create DLQ:", err)
	}
	defer dlq.Close()

	paymentConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.PaymentSucceeded,
		"order-service-payment",
		handlePaymentSucceeded,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Unable to create payment consumer:", err)
	}
	defer paymentConsumer.Close()

	inventoryConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.InventoryUnreserved,
		"order-service-inventory",
		handleInventoryUnreserved,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Unable to create inventory consumer:", err)
	}
	defer inventoryConsumer.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go relay.Start(ctx)

	limiter := middleware.NewRateLimiter(5, 2)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetOrders)
	mux.HandleFunc("GET /orders/{id}", orderHandler.GetOrder)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "order-service",
		})
	})

	appLog.Log("Listening on port "+cfg.Port, nil)
	appLog.Log("Order service started", nil)

	go paymentConsumer.Start(ctx)
	go inventoryConsumer.Start(ctx)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, appLog.Middleware(limiter.Middleware(mux))))
}
