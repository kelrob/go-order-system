package auth

import (
	"net/http"

	"github.com/kelrob/shared/middleware"
)

func RegisterRoutes(mux *http.ServeMux, handler *Handler, auth *middleware.Auth) {
	mux.HandleFunc("GET /health", handler.HealthCheck)
	mux.HandleFunc("POST /auth/signup", handler.Signup)
	mux.HandleFunc("POST /auth/login", handler.Login)

	mux.Handle("POST /auth/refresh",
		auth.Middleware(http.HandlerFunc(handler.RefreshAccessToken)),
	)

	mux.Handle("POST /auth/logout",
		auth.Middleware(http.HandlerFunc(handler.Logout)),
	)
}
