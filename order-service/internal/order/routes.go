package order

import (
	"net/http"

	"github.com/kelrob/shared/middleware"
)

func RegisterRoutes(mux *http.ServeMux, handler *Handler, auth *middleware.Auth) {
	// Public routes
	mux.HandleFunc("GET /health", http.HandlerFunc(handler.HealthCheck))

	// Protected routes
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders", handler.GetOrders)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
}
