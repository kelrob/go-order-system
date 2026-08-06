package auth

import (
	"net/http"

	"github.com/kelrob/shared/middleware"
)

func RegisterRoutes(mux *http.ServeMux, authHandler *AuthHandler, auth *middleware.Auth) {
	mux.HandleFunc("GET /health", authHandler.HealthCheck)
	mux.HandleFunc("POST /auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.RefreshAccessToken)

	mux.Handle("POST /auth/logout",
		auth.Middleware(http.HandlerFunc(authHandler.Logout)),
	)
}
