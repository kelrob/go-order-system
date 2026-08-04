package order

import "net/http"

func Register(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders", handler.GetOrders)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
	mux.HandleFunc("GET /health", handler.HealthCheck)
}
