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
	"github.com/kelrob/order-service/internal/config"
	"github.com/kelrob/order-service/internal/kafka"
	"github.com/kelrob/order-service/internal/order"
	"github.com/kelrob/shared/env"
	"github.com/kelrob/shared/events"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/middleware"
)

var (
	db     *pgxpool.Pool
	repo   *order.Repository
	appLog *logger.Logger
)

func handlePaymentSucceeded(data []byte) error {
	var event order.PaymentSucceededEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	err = repo.UpdateOrderStatus(context.Background(), event.OrderId, order.OrderStatusConfirmed)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	appLog.Info(event.TraceId, events.PaymentSucceeded, "Order confirmed", map[string]any{
		"order_id": event.OrderId,
	})
	return nil
}

func handleInventoryUnreserved(data []byte) error {
	var event order.InventoryUnreservedEvent

	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	err = repo.UpdateOrderStatus(context.Background(), event.OrderId, order.OrderStatusFailed)
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

	repo = order.NewRepository(db)
	orderService := order.NewService(repo)
	orderHandler := order.NewHandler(orderService)

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

	limiter := middleware.NewIPRateLimiter(5, 2)
	authMiddleware := middleware.NewAuth(env.Get("JWT_SECRET", "SAMPLE1$"))

	mux := http.NewServeMux()
	order.RegisterRoutes(mux, orderHandler, authMiddleware)

	appLog.Log("Listening on port "+cfg.Port, nil)
	appLog.Log("Order service started", nil)

	go paymentConsumer.Start(ctx)
	go inventoryConsumer.Start(ctx)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, appLog.Middleware(limiter.Middleware(mux))))
}
