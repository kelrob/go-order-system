package order

import (
	"net/http"

	"github.com/kelrob/shared/middleware"
)

func RegisterRoutes(mux *http.ServeMux, handler *Handler, auth *middleware.Auth) {
	// Public routes
	mux.HandleFunc("GET /health", http.HandlerFunc(handler.HealthCheck))

	// Protected routes
	mux.Handle("POST /orders",
		auth.Middleware(http.HandlerFunc(handler.CreateOrder)),
	)

	mux.Handle("GET /orders",
		auth.Middleware(http.HandlerFunc(handler.GetOrders)),
	)

	mux.Handle("GET /orders/{id}",
		auth.Middleware(http.HandlerFunc(handler.GetOrder)),
	)
}
