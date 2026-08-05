package auth

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid refresh token")
	ErrExpiredToken       = errors.New("expired refresh token")
)
