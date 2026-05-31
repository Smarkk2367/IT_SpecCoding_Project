package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("client not found")
var ErrDuplicate = errors.New("client with this name already exists")

type Client struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]Client, error) {
	const query = `
		SELECT id::text, name, created_at
		FROM clients
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clients: %w", err)
	}

	return clients, nil
}

func (r *Repository) Create(ctx context.Context, name string) (Client, error) {
	const query = `
		INSERT INTO clients (name)
		VALUES ($1)
		RETURNING id::text, name, created_at
	`

	var c Client
	err := r.db.QueryRow(ctx, query, name).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "unique constraint") {
			return Client{}, ErrDuplicate
		}
		return Client{}, fmt.Errorf("create client: %w", err)
	}

	return c, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (Client, error) {
	const query = `
		SELECT id::text, name, created_at
		FROM clients
		WHERE id = $1
	`

	var c Client
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Client{}, ErrNotFound
	}
	if err != nil {
		return Client{}, fmt.Errorf("find client by id: %w", err)
	}

	return c, nil
}
