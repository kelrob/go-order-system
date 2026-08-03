package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kelrob/auth-service/domain"
	"github.com/kelrob/shared/password"
	"github.com/kelrob/shared/ulid"
)

type UserStore interface {
	CreateUser(ctx context.Context, user domain.User) error
}

type AuthService struct {
	store UserStore
}

func NewAuthService(store UserStore) *AuthService {
	return &AuthService{store: store}
}

type CreateUserInput struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	TraceId   string
}

func (s *AuthService) CreateUser(ctx context.Context, input CreateUserInput) (domain.User, error) {
	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	user := domain.User{
		Id:             ulid.Generate(),
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Email:          input.Email,
		HashedPassword: hashedPassword,
		TraceId:        input.TraceId,
		Role:           domain.UserRole,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = s.store.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return domain.User{}, domain.ErrEmailAlreadyExists
		}
		return domain.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
