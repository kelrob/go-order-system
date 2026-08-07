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
	repo   *notification.Repository
	appLog *logger.Logger
)

func handlePaymentSucceeded(data []byte) error {
	var event notification.PaymentSucceededEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := repo.SendPaymentConfirmation(context.Background(), event); err != nil {
		return fmt.Errorf("failed to send payment confirmation: %w", err)
	}

	return nil
}

func handlerUserRegistered(data []byte) error {
	var event notification.UserRegisteredEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if err := repo.SendWelcomeEmail(context.Background(), event); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
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

	repo = notification.NewRepository(db, appLog)

	dlq, err := kafka.NewDLQ([]string{"localhost:9092"}, appLog)
	if err != nil {
		log.Fatal("Failed to create DLQ:", err)
	}
	defer dlq.Close()

	pamentSuccededConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.PaymentSucceeded,
		"notification-service-payment",
		handlePaymentSucceeded,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}
	defer pamentSuccededConsumer.Close()

	userRegisteredConsumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBroker},
		events.UserRegistered,
		"notification-service-user",
		handlerUserRegistered,
		dlq,
		appLog,
	)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}
	defer userRegisteredConsumer.Close()

	appLog.Log("Notification service started", nil)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	mux := http.NewServeMux()

	notificationHandler := notification.NewHandler(appLog)

	notification.RegisterRoutes(mux, notificationHandler)

	go func() {
		appLog.Log("Listening on port "+cfg.Port, nil)
		if err := http.ListenAndServe(":"+cfg.Port, appLog.Middleware(mux)); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	go func() {
		if err := pamentSuccededConsumer.Start(ctx); err != nil {
			log.Fatal("Consumer error:", err)
		}
	}()

	if err := userRegisteredConsumer.Start(ctx); err != nil {
		log.Fatal("Consumer error:", err)
	}
}
