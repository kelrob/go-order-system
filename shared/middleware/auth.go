package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/kelrob/shared/response"
	"github.com/kelrob/shared/ulid"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
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
		ctx := context.WithValue(r.Context(), UserIDKey, userID.String())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
