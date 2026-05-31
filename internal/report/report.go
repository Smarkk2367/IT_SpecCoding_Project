package report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("report not found")

type Report struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	RequestedBy  string     `json:"requested_by"`
	ClientID     *string    `json:"client_id,omitempty"`
	DateFrom     time.Time  `json:"date_from"`
	DateTo       time.Time  `json:"date_to"`
	FilePath     *string    `json:"file_path,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type ListFilter struct {
	Page   int
	Limit  int
	Status *string
}

type ListResult struct {
	Data  []Report
	Total int
	Page  int
}

type CreateInput struct {
	RequestedBy string
	ClientID    *string
	DateFrom    time.Time
	DateTo      time.Time
	LinkIDs     []string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (Report, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// If LinkIDs is empty but ClientID is provided, find all links belonging to client
	linkIDs := input.LinkIDs
	if len(linkIDs) == 0 && input.ClientID != nil && *input.ClientID != "" {
		const selectLinks = `SELECT id::text FROM links WHERE client_id = $1 AND deleted_at IS NULL`
		rows, err := tx.Query(ctx, selectLinks, *input.ClientID)
		if err != nil {
			return Report{}, fmt.Errorf("select client links: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var linkID string
			if err := rows.Scan(&linkID); err != nil {
				return Report{}, fmt.Errorf("scan client link: %w", err)
			}
			linkIDs = append(linkIDs, linkID)
		}
		if err := rows.Err(); err != nil {
			return Report{}, fmt.Errorf("iterate client links: %w", err)
		}
	}

	const insertReport = `
		INSERT INTO reports (requested_by, client_id, date_from, date_to, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id::text, status, requested_by::text, client_id::text, date_from, date_to, file_path, error_message, created_at, completed_at
	`

	var rep Report
	var clientID pgtype.Text
	var filePath pgtype.Text
	var errorMessage pgtype.Text
	var completedAt pgtype.Timestamptz

	err = tx.QueryRow(ctx, insertReport, input.RequestedBy, input.ClientID, input.DateFrom, input.DateTo).Scan(
		&rep.ID,
		&rep.Status,
		&rep.RequestedBy,
		&clientID,
		&rep.DateFrom,
		&rep.DateTo,
		&filePath,
		&errorMessage,
		&rep.CreatedAt,
		&completedAt,
	)
	if err != nil {
		return Report{}, fmt.Errorf("insert report: %w", err)
	}

	if clientID.Valid {
		rep.ClientID = &clientID.String
	}
	if filePath.Valid {
		rep.FilePath = &filePath.String
	}
	if errorMessage.Valid {
		rep.ErrorMessage = &errorMessage.String
	}
	if completedAt.Valid {
		rep.CompletedAt = &completedAt.Time
	}

	const insertReportLink = `
		INSERT INTO report_links (report_id, link_id)
		VALUES ($1, $2)
		ON CONFLICT (report_id, link_id) DO NOTHING
	`
	for _, linkID := range linkIDs {
		if _, err := tx.Exec(ctx, insertReportLink, rep.ID, linkID); err != nil {
			return Report{}, fmt.Errorf("insert report link %s: %w", linkID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Report{}, fmt.Errorf("commit tx: %w", err)
	}

	return rep, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	offset := (filter.Page - 1) * filter.Limit

	args := []any{}
	where := ""
	if filter.Status != nil && *filter.Status != "" {
		args = append(args, *filter.Status)
		where = "WHERE status = $1"
	}

	var total int
	countQuery := "SELECT count(*) FROM reports " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count reports: %w", err)
	}

	args = append(args, filter.Limit, offset)
	query := `
		SELECT id::text, status, requested_by::text, client_id::text, date_from, date_to, file_path, error_message, created_at, completed_at
		FROM reports
		` + where + fmt.Sprintf(`
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	reports := make([]Report, 0)
	for rows.Next() {
		var rep Report
		var clientID pgtype.Text
		var filePath pgtype.Text
		var errorMessage pgtype.Text
		var completedAt pgtype.Timestamptz

		err := rows.Scan(
			&rep.ID,
			&rep.Status,
			&rep.RequestedBy,
			&clientID,
			&rep.DateFrom,
			&rep.DateTo,
			&filePath,
			&errorMessage,
			&rep.CreatedAt,
			&completedAt,
		)
		if err != nil {
			return ListResult{}, fmt.Errorf("scan report: %w", err)
		}

		if clientID.Valid && clientID.String != "" {
			rep.ClientID = &clientID.String
		}
		if filePath.Valid {
			rep.FilePath = &filePath.String
		}
		if errorMessage.Valid {
			rep.ErrorMessage = &errorMessage.String
		}
		if completedAt.Valid {
			rep.CompletedAt = &completedAt.Time
		}

		reports = append(reports, rep)
	}

	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate reports: %w", err)
	}

	return ListResult{Data: reports, Total: total, Page: filter.Page}, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (Report, error) {
	const query = `
		SELECT id::text, status, requested_by::text, client_id::text, date_from, date_to, file_path, error_message, created_at, completed_at
		FROM reports
		WHERE id = $1
	`

	var rep Report
	var clientID pgtype.Text
	var filePath pgtype.Text
	var errorMessage pgtype.Text
	var completedAt pgtype.Timestamptz

	err := r.db.QueryRow(ctx, query, id).Scan(
		&rep.ID,
		&rep.Status,
		&rep.RequestedBy,
		&clientID,
		&rep.DateFrom,
		&rep.DateTo,
		&filePath,
		&errorMessage,
		&rep.CreatedAt,
		&completedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrNotFound
	}
	if err != nil {
		return Report{}, fmt.Errorf("find report: %w", err)
	}

	if clientID.Valid && clientID.String != "" {
		rep.ClientID = &clientID.String
	}
	if filePath.Valid {
		rep.FilePath = &filePath.String
	}
	if errorMessage.Valid {
		rep.ErrorMessage = &errorMessage.String
	}
	if completedAt.Valid {
		rep.CompletedAt = &completedAt.Time
	}

	return rep, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status string, filePath *string, errorMessage *string) error {
	const query = `
		UPDATE reports
		SET status = $2, file_path = $3, error_message = $4, completed_at = $5
		WHERE id = $1
	`

	var completedAt *time.Time
	if status == "done" || status == "failed" {
		now := time.Now().UTC()
		completedAt = &now
	}

	tag, err := r.db.Exec(ctx, query, id, status, filePath, errorMessage, completedAt)
	if err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
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

type LinkReportData struct {
	ID           string
	ShortCode    string
	OriginalURL  string
	CampaignName string
	TotalClicks  int
	UniqueClicks int
}

type ReportDataset struct {
	ReportID       string
	ClientName     string
	DateFrom       time.Time
	DateTo         time.Time
	TotalClicks    int
	UniqueClicks   int
	Links          []LinkReportData
	TopCountries   []CountryCount
	TopDevices     []DeviceCount
	TopReferrers   []ReferrerCount
	ClicksOverTime []TimeCount
}

func (r *Repository) GetReportDataset(ctx context.Context, id string) (ReportDataset, error) {
	rep, err := r.FindByID(ctx, id)
	if err != nil {
		return ReportDataset{}, err
	}

	var clientName string
	if rep.ClientID != nil {
		const queryClient = `SELECT name FROM clients WHERE id = $1`
		err = r.db.QueryRow(ctx, queryClient, *rep.ClientID).Scan(&clientName)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return ReportDataset{}, fmt.Errorf("query client name: %w", err)
		}
	}

	dataset := ReportDataset{
		ReportID:   rep.ID,
		ClientName: clientName,
		DateFrom:   rep.DateFrom,
		DateTo:     rep.DateTo,
	}

	// 1. Total and Unique Clicks
	const queryTotal = `
		SELECT count(*), count(DISTINCT ip_hash)
		FROM clicks
		WHERE link_id IN (SELECT link_id FROM report_links WHERE report_id = $1)
		  AND clicked_at >= $2
		  AND clicked_at <= $3
	`
	err = r.db.QueryRow(ctx, queryTotal, rep.ID, rep.DateFrom, rep.DateTo).Scan(&dataset.TotalClicks, &dataset.UniqueClicks)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query total clicks: %w", err)
	}

	// 2. Clicks over time (grouped by day)
	const queryOverTime = `
		SELECT date_trunc('day', clicked_at) AS bucket, count(*)
		FROM clicks
		WHERE link_id IN (SELECT link_id FROM report_links WHERE report_id = $1)
		  AND clicked_at >= $2
		  AND clicked_at <= $3
		GROUP BY bucket
		ORDER BY bucket
	`
	rows, err := r.db.Query(ctx, queryOverTime, rep.ID, rep.DateFrom, rep.DateTo)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query clicks over time: %w", err)
	}
	defer rows.Close()

	dataset.ClicksOverTime = make([]TimeCount, 0)
	for rows.Next() {
		var tc TimeCount
		if err := rows.Scan(&tc.Timestamp, &tc.Count); err != nil {
			return ReportDataset{}, fmt.Errorf("scan clicks over time: %w", err)
		}
		dataset.ClicksOverTime = append(dataset.ClicksOverTime, tc)
	}

	// 3. Top Countries
	const queryCountries = `
		SELECT country, count(*)
		FROM clicks
		WHERE link_id IN (SELECT link_id FROM report_links WHERE report_id = $1)
		  AND clicked_at >= $2
		  AND clicked_at <= $3
		  AND country IS NOT NULL
		GROUP BY country
		ORDER BY count(*) DESC
		LIMIT 5
	`
	rowsC, err := r.db.Query(ctx, queryCountries, rep.ID, rep.DateFrom, rep.DateTo)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query top countries: %w", err)
	}
	defer rowsC.Close()

	dataset.TopCountries = make([]CountryCount, 0)
	for rowsC.Next() {
		var cc CountryCount
		if err := rowsC.Scan(&cc.Country, &cc.Count); err != nil {
			return ReportDataset{}, fmt.Errorf("scan top country: %w", err)
		}
		dataset.TopCountries = append(dataset.TopCountries, cc)
	}

	// 4. Top Devices
	const queryDevices = `
		SELECT device_type, count(*)
		FROM clicks
		WHERE link_id IN (SELECT link_id FROM report_links WHERE report_id = $1)
		  AND clicked_at >= $2
		  AND clicked_at <= $3
		  AND device_type IS NOT NULL
		GROUP BY device_type
		ORDER BY count(*) DESC
		LIMIT 5
	`
	rowsD, err := r.db.Query(ctx, queryDevices, rep.ID, rep.DateFrom, rep.DateTo)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query top devices: %w", err)
	}
	defer rowsD.Close()

	dataset.TopDevices = make([]DeviceCount, 0)
	for rowsD.Next() {
		var dc DeviceCount
		if err := rowsD.Scan(&dc.DeviceType, &dc.Count); err != nil {
			return ReportDataset{}, fmt.Errorf("scan top device: %w", err)
		}
		dataset.TopDevices = append(dataset.TopDevices, dc)
	}

	// 5. Top Referrers
	const queryReferrers = `
		SELECT referrer, count(*)
		FROM clicks
		WHERE link_id IN (SELECT link_id FROM report_links WHERE report_id = $1)
		  AND clicked_at >= $2
		  AND clicked_at <= $3
		  AND referrer IS NOT NULL
		GROUP BY referrer
		ORDER BY count(*) DESC
		LIMIT 5
	`
	rowsR, err := r.db.Query(ctx, queryReferrers, rep.ID, rep.DateFrom, rep.DateTo)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query top referrers: %w", err)
	}
	defer rowsR.Close()

	dataset.TopReferrers = make([]ReferrerCount, 0)
	for rowsR.Next() {
		var rc ReferrerCount
		if err := rowsR.Scan(&rc.Referrer, &rc.Count); err != nil {
			return ReportDataset{}, fmt.Errorf("scan top referrer: %w", err)
		}
		dataset.TopReferrers = append(dataset.TopReferrers, rc)
	}

	// 6. Links List
	const queryLinks = `
		SELECT l.id::text, l.short_code, l.original_url, COALESCE(l.campaign_name, ''),
		       (SELECT count(*) FROM clicks c WHERE c.link_id = l.id AND c.clicked_at >= $2 AND c.clicked_at <= $3) AS total_clicks,
		       (SELECT count(DISTINCT c.ip_hash) FROM clicks c WHERE c.link_id = l.id AND c.clicked_at >= $2 AND c.clicked_at <= $3) AS unique_clicks
		FROM links l
		JOIN report_links rl ON rl.link_id = l.id
		WHERE rl.report_id = $1
	`
	rowsL, err := r.db.Query(ctx, queryLinks, rep.ID, rep.DateFrom, rep.DateTo)
	if err != nil {
		return ReportDataset{}, fmt.Errorf("query links data: %w", err)
	}
	defer rowsL.Close()

	dataset.Links = make([]LinkReportData, 0)
	for rowsL.Next() {
		var ld LinkReportData
		if err := rowsL.Scan(&ld.ID, &ld.ShortCode, &ld.OriginalURL, &ld.CampaignName, &ld.TotalClicks, &ld.UniqueClicks); err != nil {
			return ReportDataset{}, fmt.Errorf("scan link report data: %w", err)
		}
		dataset.Links = append(dataset.Links, ld)
	}

	return dataset, nil
}

func (r *Repository) FindUserEmail(ctx context.Context, userID string) (string, error) {
	const query = `SELECT email FROM users WHERE id = $1`
	var email string
	if err := r.db.QueryRow(ctx, query, userID).Scan(&email); err != nil {
		return "", fmt.Errorf("query user email: %w", err)
	}
	return email, nil
}

func (r *Repository) GetReportLinkIDs(ctx context.Context, reportID string) ([]string, error) {
	const query = `SELECT link_id::text FROM report_links WHERE report_id = $1`
	rows, err := r.db.Query(ctx, query, reportID)
	if err != nil {
		return nil, fmt.Errorf("query report links: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan report link: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
