package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/auth-service/config"
	"github.com/kelrob/shared/logger"
)

var (
	db     *pgxpool.Pool
	appLog *logger.Logger
	err    error
)

func main() {
	appLog = logger.NewLogger("auth-service")
	cfg := config.Load()

	db, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		appLog.Fatal("Unable to connect to database", err)
	}
	defer db.Close()

	err = db.Ping(context.Background())
	if err != nil {
		appLog.Fatal("Unable to ping database", err)
	}

	appLog.Log("Connected to database successfully", nil)

	mux := http.NewServeMux()
	appLog.Log("Listening on port "+cfg.Port, nil)
	appLog.Log("Auth service started", nil)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, appLog.Middleware(mux)))
}
