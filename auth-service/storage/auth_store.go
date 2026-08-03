package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/auth-service/domain"
	"github.com/kelrob/shared/events"
)

const pgUniqueViolation = "23505"

type AuthStore struct {
	db *pgxpool.Pool
}

func NewAuthStore(db *pgxpool.Pool) *AuthStore {
	return &AuthStore{db: db}
}

func (a *AuthStore) CreateUser(ctx context.Context, user domain.User) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transactions: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, first_name, last_name, hashed_password, role, is_locked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.Id,
		user.Email,
		user.FirstName,
		user.LastName,
		user.HashedPassword,
		user.Role,
		user.IsLocked,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		tx.Rollback(ctx)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return domain.ErrEmailAlreadyExists
		}

		return fmt.Errorf("failed to insert user: %w", err)
	}

	payload, err := json.Marshal(domain.UserRegisteredEvent{
		TraceId:   user.TraceId,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
	})
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload)
		VALUES ($1, $2, $3)`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		events.UserRegistered,
		payload,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert outbox: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
