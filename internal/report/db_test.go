package report

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryIntegration(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL and DATABASE_URL are not set, skipping database integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Failed to connect to database: %v. Skipping integration test.", err)
	}
	defer pool.Close()

	// Verify we can ping the database
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Database is not reachable: %v. Skipping integration test.", err)
	}

	// Create repository
	repo := NewRepository(pool)

	// Set up unique name suffix
	uniqueSuffix := time.Now().Format("20060102150405.000000")

	// Set up test client
	var clientID string
	err = pool.QueryRow(ctx, "INSERT INTO clients (name) VALUES ($1) RETURNING id::text", "Test Integration Client "+uniqueSuffix).Scan(&clientID)
	if err != nil {
		t.Fatalf("failed to insert test client: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM clients WHERE id = $1", clientID)
	}()

	// Set up test user (marketer)
	var userID string
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'marketer')
		RETURNING id::text
	`, "marketer-"+uniqueSuffix+"@test.com", "hash").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	}()

	// Set up test links
	var link1ID string
	err = pool.QueryRow(ctx, `
		INSERT INTO links (short_code, original_url, created_by, campaign_name, client_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, "test1-"+uniqueSuffix, "https://example.com/1", userID, "Spring 2026", clientID).Scan(&link1ID)
	if err != nil {
		t.Fatalf("failed to insert link 1: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM links WHERE id = $1", link1ID)
	}()

	var link2ID string
	err = pool.QueryRow(ctx, `
		INSERT INTO links (short_code, original_url, created_by, campaign_name, client_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, "test2-"+uniqueSuffix, "https://example.com/2", userID, "Summer 2026", clientID).Scan(&link2ID)
	if err != nil {
		t.Fatalf("failed to insert link 2: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM links WHERE id = $1", link2ID)
	}()

	// Let's insert some clicks
	now := time.Now().UTC()
	clicksData := []struct {
		linkID     string
		clickedAt  time.Time
		country    string
		deviceType string
		referrer   string
		ipHash     string
		eventID    string
	}{
		{link1ID, now.Add(-10 * time.Minute), "PL", "desktop", "google.com", "ip1", "00000000-0000-0000-0000-" + uniqueSuffix[:12]},
		{link1ID, now.Add(-5 * time.Minute), "PL", "mobile", "facebook.com", "ip2", "00000000-0000-0000-0001-" + uniqueSuffix[:12]},
		{link2ID, now.Add(-1 * time.Minute), "US", "desktop", "direct", "ip1", "00000000-0000-0000-0002-" + uniqueSuffix[:12]}, // same ip_hash as first to test unique clicks
	}

	for _, click := range clicksData {
		_, err := pool.Exec(ctx, `
			INSERT INTO clicks (link_id, clicked_at, country, device_type, referrer, ip_hash, event_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, click.linkID, click.clickedAt, click.country, click.deviceType, click.referrer, click.ipHash, click.eventID)
		if err != nil {
			t.Fatalf("failed to insert click: %v", err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM clicks WHERE link_id IN ($1, $2)", link1ID, link2ID)
	}()

	// Test FindUserEmail
	email, err := repo.FindUserEmail(ctx, userID)
	if err != nil {
		t.Fatalf("FindUserEmail failed: %v", err)
	}
	if !strings.Contains(email, "marketer-") {
		t.Errorf("expected email to contain marketer-, got %s", email)
	}

	// Test Create Report with LinkIDs
	input := CreateInput{
		RequestedBy: userID,
		ClientID:    &clientID,
		DateFrom:    now.Add(-1 * time.Hour),
		DateTo:      now.Add(1 * time.Hour),
		LinkIDs:     []string{link1ID, link2ID},
	}

	report, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create report failed: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reports WHERE id = $1", report.ID)
	}()

	if report.Status != "pending" {
		t.Errorf("expected report status pending, got %s", report.Status)
	}
	if report.ClientID == nil || *report.ClientID != clientID {
		t.Errorf("expected client ID %s, got %v", clientID, report.ClientID)
	}

	// Test GetReportLinkIDs
	linkIDs, err := repo.GetReportLinkIDs(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetReportLinkIDs failed: %v", err)
	}
	if len(linkIDs) != 2 {
		t.Errorf("expected 2 link IDs, got %d", len(linkIDs))
	}

	// Test FindByID
	foundReport, err := repo.FindByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if foundReport.ID != report.ID {
		t.Errorf("expected report ID %s, got %s", report.ID, foundReport.ID)
	}

	// Test List
	filter := ListFilter{
		Page:  1,
		Limit: 10,
	}
	listRes, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List reports failed: %v", err)
	}
	if listRes.Total == 0 {
		t.Error("expected at least 1 report in list")
	}

	// Test UpdateStatus
	newFilePath := "/data/reports/report_" + report.ID + ".pdf"
	err = repo.UpdateStatus(ctx, report.ID, "done", &newFilePath, nil)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	updatedReport, err := repo.FindByID(ctx, report.ID)
	if err != nil {
		t.Fatalf("FindByID after update failed: %v", err)
	}
	if updatedReport.Status != "done" {
		t.Errorf("expected status done, got %s", updatedReport.Status)
	}
	if updatedReport.FilePath == nil || *updatedReport.FilePath != newFilePath {
		t.Errorf("expected filepath %s, got %v", newFilePath, updatedReport.FilePath)
	}

	// Test GetReportDataset & calculation
	dataset, err := repo.GetReportDataset(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetReportDataset failed: %v", err)
	}

	if !strings.HasPrefix(dataset.ClientName, "Test Integration Client") {
		t.Errorf("expected ClientName to start with Test Integration Client, got %s", dataset.ClientName)
	}
	if dataset.TotalClicks != 3 {
		t.Errorf("expected 3 total clicks, got %d", dataset.TotalClicks)
	}
	if dataset.UniqueClicks != 2 {
		t.Errorf("expected 2 unique clicks (ip1 occurred twice), got %d", dataset.UniqueClicks)
	}

	// Verify TopCountries
	if len(dataset.TopCountries) != 2 {
		t.Errorf("expected 2 top countries, got %d", len(dataset.TopCountries))
	} else {
		if dataset.TopCountries[0].Country != "PL" || dataset.TopCountries[0].Count != 2 {
			t.Errorf("expected top country PL with count 2, got %v", dataset.TopCountries[0])
		}
		if dataset.TopCountries[1].Country != "US" || dataset.TopCountries[1].Count != 1 {
			t.Errorf("expected second country US with count 1, got %v", dataset.TopCountries[1])
		}
	}

	// Verify TopDevices
	if len(dataset.TopDevices) != 2 {
		t.Errorf("expected 2 top devices, got %d", len(dataset.TopDevices))
	}

	// Verify TopReferrers
	if len(dataset.TopReferrers) != 3 {
		t.Errorf("expected 3 top referrers, got %d", len(dataset.TopReferrers))
	}

	// Verify ClicksOverTime
	if len(dataset.ClicksOverTime) == 0 {
		t.Error("expected clicks over time details")
	}

	// Verify Links list
	if len(dataset.Links) != 2 {
		t.Errorf("expected 2 links, got %d", len(dataset.Links))
	}
}
