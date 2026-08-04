package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/kelrob/shared/password"
	"github.com/kelrob/shared/ulid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (RefreshToken, error)
}

type AuthService struct {
	repo            UserRepository
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthService(repo UserRepository, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:            repo,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

type CreateUserInput struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	TraceId   string
}

type LoginInput struct {
	Email    string
	Password string
	TraceId  string
}

func (s *AuthService) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	user := User{
		Id:             ulid.Generate(),
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		HashedPassword: hashedPassword,
		TraceId:        input.TraceId,
		Role:           UserRole,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			return User{}, ErrEmailAlreadyExists
		}
		return User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	if !password.Verify(input.Password, user.HashedPassword) {
		return LoginResponse{}, ErrInvalidCredentials
	}

	accessToken, err := s.generateAccessToken(&user)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	now := time.Now()
	err = s.repo.CreateRefreshToken(ctx, RefreshToken{
		Id:        ulid.Generate(),
		UserId:    user.Id,
		Token:     refreshToken,
		ExpiresAt: now.Add(s.refreshTokenTTL),
		CreatedAt: now,
	})
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	token, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", ErrInvalidToken
	}

	if token.Revoked {
		return "", ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		return "", ErrExpiredToken
	}

	user, err := s.repo.GetUserByID(ctx, token.UserId)
	if err != nil {
		return "", err
	}

	accessToken, err := s.generateAccessToken(&user)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func (s *AuthService) generateAccessToken(user *User) (string, error) {
	expirationTime := time.Now().Add(s.accessTokenTTL)

	claims := jwt.MapClaims{
		"sub":   user.Id,
		"email": user.Email,
		"role":  user.Role,
		"exp":   expirationTime.Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (s *AuthService) generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
