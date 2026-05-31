package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type Query struct {
	LinkID   string
	Period   string
	DateFrom *time.Time
	DateTo   *time.Time
}

type Result struct {
	TotalClicks    int
	UniqueClicks   int
	ClicksOverTime []TimeCount
	ByCountry      []CountryCount
	ByDevice       []DeviceCount
	ByReferrer     []ReferrerCount
}

type TimeCount struct {
	Timestamp time.Time
	Count     int
}

type CountryCount struct {
	Country string
	Count   int
}

type DeviceCount struct {
	DeviceType string
	Count      int
}

type ReferrerCount struct {
	Referrer string
	Count    int
}

func (r *Repository) Get(ctx context.Context, query Query) (Result, error) {
	period := normalizePeriod(query.Period)
	args := []any{query.LinkID}
	where := "WHERE link_id = $1"
	if query.DateFrom != nil {
		args = append(args, *query.DateFrom)
		where += fmt.Sprintf(" AND clicked_at >= $%d", len(args))
	}
	if query.DateTo != nil {
		args = append(args, *query.DateTo)
		where += fmt.Sprintf(" AND clicked_at <= $%d", len(args))
	}

	var result Result
	countQuery := "SELECT count(*), count(DISTINCT ip_hash) FROM clicks " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&result.TotalClicks, &result.UniqueClicks); err != nil {
		return Result{}, fmt.Errorf("count stats: %w", err)
	}

	timeRows, err := r.db.Query(ctx, `
		SELECT date_trunc('`+period+`', clicked_at) AS bucket, count(*)
		FROM clicks
		`+where+`
		GROUP BY bucket
		ORDER BY bucket
	`, args...)
	if err != nil {
		return Result{}, fmt.Errorf("time stats: %w", err)
	}
	defer timeRows.Close()
	for timeRows.Next() {
		var row TimeCount
		if err := timeRows.Scan(&row.Timestamp, &row.Count); err != nil {
			return Result{}, fmt.Errorf("scan time stats: %w", err)
		}
		result.ClicksOverTime = append(result.ClicksOverTime, row)
	}

	countries, err := r.top(ctx, "country", where+" AND country IS NOT NULL", args)
	if err != nil {
		return Result{}, err
	}
	for _, row := range countries {
		result.ByCountry = append(result.ByCountry, CountryCount{Country: row.Label, Count: row.Count})
	}

	devices, err := r.top(ctx, "device_type", where+" AND device_type IS NOT NULL", args)
	if err != nil {
		return Result{}, err
	}
	for _, row := range devices {
		result.ByDevice = append(result.ByDevice, DeviceCount{DeviceType: row.Label, Count: row.Count})
	}

	referrers, err := r.top(ctx, "referrer", where+" AND referrer IS NOT NULL", args)
	if err != nil {
		return Result{}, err
	}
	for _, row := range referrers {
		result.ByReferrer = append(result.ByReferrer, ReferrerCount{Referrer: row.Label, Count: row.Count})
	}

	return result, nil
}

type topRow struct {
	Label string
	Count int
}

func (r *Repository) top(ctx context.Context, column string, where string, args []any) ([]topRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+column+`, count(*)
		FROM clicks
		`+where+`
		GROUP BY `+column+`
		ORDER BY count(*) DESC
		LIMIT 5
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("top %s stats: %w", column, err)
	}
	defer rows.Close()

	result := make([]topRow, 0)
	for rows.Next() {
		var row topRow
		if err := rows.Scan(&row.Label, &row.Count); err != nil {
			return nil, fmt.Errorf("scan top %s stats: %w", column, err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func normalizePeriod(period string) string {
	switch period {
	case "hour", "day", "week":
		return period
	default:
		return "day"
	}
}
