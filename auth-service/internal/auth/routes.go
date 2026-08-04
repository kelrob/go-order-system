package auth

import (
	"net/http"
)

func Register(mux *http.ServeMux, authHandler *AuthHandler) {
	mux.HandleFunc("GET /health", authHandler.HealthCheck)
	mux.HandleFunc("POST /auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
}
