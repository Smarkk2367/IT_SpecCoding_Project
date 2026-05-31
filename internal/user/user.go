package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	ClientID     *string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id::text, email, password_hash, role, COALESCE(client_id::text, '')
		FROM users
		WHERE email = $1
	`

	return r.queryOne(ctx, query, email)
}

func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	const query = `
		SELECT id::text, email, password_hash, role, COALESCE(client_id::text, '')
		FROM users
		WHERE id = $1
	`

	return r.queryOne(ctx, query, id)
}

func (r *Repository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	const query = `UPDATE users SET password_hash = $2 WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) queryOne(ctx context.Context, query string, args ...any) (User, error) {
	var result User
	var clientID string
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&result.ID,
		&result.Email,
		&result.PasswordHash,
		&result.Role,
		&clientID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}
	if clientID != "" {
		result.ClientID = &clientID
	}

	return result, nil
}
