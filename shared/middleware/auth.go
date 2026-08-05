package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/kelrob/shared/response"
	"github.com/kelrob/shared/ulid"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
	RoleKey   contextKey = "role"
)

type Auth struct {
	jwtSecret []byte
}

var (
	ErrInvalidToken = errors.New("invalid refresh token")
	ErrExpiredToken = errors.New("expired refresh token")
)

func NewAuth(jwtSecret string) *Auth {
	return &Auth{jwtSecret: []byte(jwtSecret)}
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "missing authorization header", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(w, http.StatusUnauthorized, "invalid authorization format", nil)
			return
		}

		tokenString := parts[1]
		claims, err := a.validateJWToken(tokenString)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token", nil)
			return
		}

		userIDStr, ok := claims["sub"].(string)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "invalid token claims", nil)
			return
		}

		userID, err := ulid.Parse(userIDStr)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid user ID in token", nil)
			return
		}

		role, _ := claims["role"].(string)

		ctx := context.WithValue(r.Context(), UserIDKey, userID.String())
		ctx = context.WithValue(ctx, RoleKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentRole, _ := r.Context().Value(RoleKey).(string)

			fmt.Println("Current Role:", currentRole)
			fmt.Println("Required Role:", role)

			if currentRole != role {
				response.Error(w, http.StatusForbidden, "forbidden", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (a *Auth) validateJWToken(jwtToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken

}
