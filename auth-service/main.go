package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/auth-service/config"
	"github.com/kelrob/auth-service/handlers"
	"github.com/kelrob/auth-service/service"
	"github.com/kelrob/auth-service/storage"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/middleware"
	"github.com/kelrob/shared/response"
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

	authStore := storage.NewAuthStore(db)
	authService := service.NewAuthService(authStore)
	authHandler := handlers.NewAuthHandler(authService)

	limiter := middleware.NewRateLimiter(5, 2)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/signup", authHandler.CreateAccount)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "auth-service",
		})
	})

	appLog.Log("Listening on port "+cfg.Port, nil)
	appLog.Log("Auth service started", nil)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, appLog.Middleware(limiter.Middleware(mux))))
}
