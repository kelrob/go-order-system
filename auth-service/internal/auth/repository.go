package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelrob/shared/events"
)

const pgUniqueViolation = "23505"

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (a *AuthRepository) CreateUser(ctx context.Context, user User) error {
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
			return ErrEmailAlreadyExists
		}

		return fmt.Errorf("failed to insert user: %w", err)
	}

	payload, err := json.Marshal(UserRegisteredEvent{
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

func (a *AuthRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User

	err := a.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, hashed_password, role, is_locked, created_at, updated_at
		FROM users WHERE email = $1`,
		strings.TrimSpace(email),
	).Scan(
		&user.Id,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.HashedPassword,
		&user.Role,
		&user.IsLocked,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (a *AuthRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) error {
	_, err := a.db.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		token.Id,
		token.UserId,
		token.Token,
		token.ExpiresAt,
		token.CreatedAt,
		token.Revoked,
	)
	if err != nil {
		return fmt.Errorf("failed to insert refresh token: %w", err)
	}

	return nil
}

func (a *AuthRepository) RotateRefreshToken(ctx context.Context, oldToken string, newToken RefreshToken) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE token = $1`,
		oldToken,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		newToken.Id,
		newToken.UserId,
		newToken.Token,
		newToken.ExpiresAt,
		newToken.CreatedAt,
		newToken.Revoked,
	)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert new refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (a *AuthRepository) GetRefreshToken(ctx context.Context, refreshToken string) (RefreshToken, error) {
	var token RefreshToken

	err := a.db.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at, created_at, revoked
		FROM refresh_tokens WHERE token = $1`,
		strings.TrimSpace(refreshToken),
	).Scan(
		&token.Id,
		&token.UserId,
		&token.Token,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.Revoked,
	)

	if err != nil {
		return RefreshToken{}, err
	}

	return token, nil
}

func (a *AuthRepository) GetUserByID(ctx context.Context, id string) (User, error) {
	var user User

	err := a.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, role, is_locked, created_at, updated_at
		FROM users WHERE id = $1`,
		strings.TrimSpace(id),
	).Scan(
		&user.Id,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsLocked,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (a *AuthRepository) UpdateRefreshTokenToExpired(ctx context.Context, userID string) error {
	_, err := a.db.Query(ctx, `UPDATE refresh_tokens
        SET revoked = true
        WHERE user_id = $1`, userID)

	return err
}
