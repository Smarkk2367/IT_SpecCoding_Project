package link

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")

type Link struct {
	ID           string
	ShortCode    string
	OriginalURL  string
	CampaignName *string
	ClientID     *string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
}

type ListFilter struct {
	Page         int
	Limit        int
	ClientID     *string
	CampaignName *string
	Active       *bool
}

type ListResult struct {
	Data  []Link
	Total int
	Page  int
}

type CreateInput struct {
	OriginalURL  string
	CampaignName *string
	ClientID     *string
	ExpiresAt    *time.Time
	CreatedBy    string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindRedirectByShortCode(ctx context.Context, shortCode string) (Link, error) {
	const query = `
		SELECT id::text, short_code, original_url, expires_at
		FROM links
		WHERE short_code = $1
		  AND is_active = true
		  AND deleted_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1
	`

	var result Link
	var expiresAt pgtype.Timestamptz
	err := r.db.QueryRow(ctx, query, shortCode).Scan(
		&result.ID,
		&result.ShortCode,
		&result.OriginalURL,
		&expiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("query redirect link: %w", err)
	}
	if expiresAt.Valid {
		result.ExpiresAt = &expiresAt.Time
	}

	return result, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	filter = normalizeListFilter(filter)
	offset := (filter.Page - 1) * filter.Limit

	args := []any{}
	where := "WHERE deleted_at IS NULL"
	if filter.ClientID != nil {
		args = append(args, *filter.ClientID)
		where += fmt.Sprintf(" AND client_id = $%d", len(args))
	}
	if filter.CampaignName != nil {
		args = append(args, *filter.CampaignName)
		where += fmt.Sprintf(" AND campaign_name = $%d", len(args))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}

	var total int
	countQuery := "SELECT count(*) FROM links " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count links: %w", err)
	}

	args = append(args, filter.Limit, offset)
	query := `
		SELECT id::text, short_code, original_url, COALESCE(campaign_name, ''), COALESCE(client_id::text, ''), expires_at, created_at
		FROM links
		` + where + fmt.Sprintf(`
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()

	links := make([]Link, 0)
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return ListResult{}, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate links: %w", err)
	}

	return ListResult{Data: links, Total: total, Page: filter.Page}, nil
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (Link, error) {
	shortCode, err := generateShortCode()
	if err != nil {
		return Link{}, err
	}

	const query = `
		INSERT INTO links (short_code, original_url, created_by, campaign_name, client_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, short_code, original_url, COALESCE(campaign_name, ''), COALESCE(client_id::text, ''), expires_at, created_at
	`

	var result Link
	for attempts := 0; attempts < 5; attempts++ {
		result, err = scanLink(r.db.QueryRow(ctx, query, shortCode, input.OriginalURL, input.CreatedBy, input.CampaignName, input.ClientID, input.ExpiresAt))
		if err == nil {
			return result, nil
		}
		if !isUniqueViolation(err) {
			return Link{}, fmt.Errorf("create link: %w", err)
		}
		shortCode, err = generateShortCode()
		if err != nil {
			return Link{}, err
		}
	}

	return Link{}, errors.New("could not generate unique short code")
}

func (r *Repository) FindByID(ctx context.Context, id string) (Link, error) {
	const query = `
		SELECT id::text, short_code, original_url, COALESCE(campaign_name, ''), COALESCE(client_id::text, ''), expires_at, created_at
		FROM links
		WHERE id = $1 AND deleted_at IS NULL
	`

	var result Link
	result, err := scanLink(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("find link: %w", err)
	}

	return result, nil
}

func (r *Repository) Delete(ctx context.Context, id string) (Link, error) {
	const query = `
		UPDATE links
		SET deleted_at = now(), is_active = false
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id::text, short_code, original_url, COALESCE(campaign_name, ''), COALESCE(client_id::text, ''), expires_at, created_at
	`

	var result Link
	result, err := scanLink(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("delete link: %w", err)
	}

	return result, nil
}

func normalizeListFilter(filter ListFilter) ListFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return filter
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLink(row rowScanner) (Link, error) {
	var result Link
	var campaignName string
	var clientID string
	var expiresAt pgtype.Timestamptz

	if err := row.Scan(
		&result.ID,
		&result.ShortCode,
		&result.OriginalURL,
		&campaignName,
		&clientID,
		&expiresAt,
		&result.CreatedAt,
	); err != nil {
		return Link{}, err
	}

	if campaignName != "" {
		result.CampaignName = &campaignName
	}
	if clientID != "" {
		result.ClientID = &clientID
	}
	if expiresAt.Valid {
		result.ExpiresAt = &expiresAt.Time
	}

	return result, nil
}

func generateShortCode() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	bytes := make([]byte, length)
	for i := range bytes {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate short code: %w", err)
		}
		bytes[i] = alphabet[n.Int64()]
	}
	return string(bytes), nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value")
}
