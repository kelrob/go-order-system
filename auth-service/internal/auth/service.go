package auth

import (
	"context"
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
}

type AuthService struct {
	repo           UserRepository
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

func NewAuthService(repo UserRepository, jwtSecret string, accessTokenTTL time.Duration) *AuthService {
	return &AuthService{
		repo:           repo,
		jwtSecret:      []byte(jwtSecret),
		accessTokenTTL: accessTokenTTL,
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

func (s *AuthService) Login(ctx context.Context, input LoginInput) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if !password.Verify(input.Password, user.HashedPassword) {
		return "", ErrInvalidCredentials
	}

	token, err := s.generateAccessToken(&user)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, nil
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
