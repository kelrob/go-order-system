package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/auth-service/internal/auth"
	"github.com/kelrob/auth-service/internal/config"
	"github.com/kelrob/shared/env"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/middleware"
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

	jwtSecret := env.Get("JWT_SECRET", "SAMPLE1$")

	authRepo := auth.NewAuthRepository(db)
	authService := auth.NewAuthService(authRepo, jwtSecret, 15*time.Minute, 7*24*time.Hour)
	authHandler := auth.NewHandler(authService, appLog)

	limiter := middleware.NewIPRateLimiter(5, 2)
	authMiddleware := middleware.NewAuth(jwtSecret)

	mux := http.NewServeMux()
	auth.RegisterRoutes(mux, authHandler, authMiddleware)
	appLog.Log("Listening on port "+cfg.Port, nil)
	appLog.Log("Auth service started", nil)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, appLog.Middleware(limiter.Middleware(mux))))
}
