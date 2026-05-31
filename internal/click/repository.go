package click

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicate = errors.New("duplicate click event")

type Click struct {
	LinkID     string
	ClickedAt  time.Time
	Country    *string
	City       *string
	DeviceType *string
	Browser    *string
	OS         *string
	Referrer   *string
	IPHash     string
	EventID    string
	UserAgent  string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ExistsEvent(ctx context.Context, eventID string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM clicks WHERE event_id = $1)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, eventID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check click event: %w", err)
	}
	return exists, nil
}

func (r *Repository) Insert(ctx context.Context, click Click) error {
	const query = `
		INSERT INTO clicks (
			link_id,
			clicked_at,
			country,
			city,
			device_type,
			browser,
			os,
			referrer,
			ip_hash,
			event_id,
			user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (event_id) DO NOTHING
	`

	tag, err := r.db.Exec(
		ctx,
		query,
		click.LinkID,
		click.ClickedAt,
		click.Country,
		click.City,
		click.DeviceType,
		click.Browser,
		click.OS,
		click.Referrer,
		click.IPHash,
		click.EventID,
		click.UserAgent,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDuplicate
		}
		return fmt.Errorf("insert click: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDuplicate
	}

	return nil
}
