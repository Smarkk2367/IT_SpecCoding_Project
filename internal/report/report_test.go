package report

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestReportTemplateRendering(t *testing.T) {
	tmpl, err := GetTemplate()
	if err != nil {
		t.Fatalf("failed to compile template: %v", err)
	}

	dateFrom := time.Now().Add(-7 * 24 * time.Hour).UTC()
	dateTo := time.Now().UTC()

	dataset := ReportDataset{
		ReportID:     "test-report-uuid-12345",
		ClientName:   "Test Client Inc.",
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		TotalClicks:  120,
		UniqueClicks: 85,
		Links: []LinkReportData{
			{
				ID:           "link-1",
				ShortCode:    "abcde",
				OriginalURL:  "https://google.com",
				CampaignName: "Spring Sale",
				TotalClicks:  80,
				UniqueClicks: 60,
			},
			{
				ID:           "link-2",
				ShortCode:    "fghij",
				OriginalURL:  "https://yahoo.com",
				CampaignName: "Newsletter",
				TotalClicks:  40,
				UniqueClicks: 25,
			},
		},
		TopCountries: []CountryCount{
			{Country: "PL", Count: 100},
			{Country: "US", Count: 20},
		},
		TopDevices: []DeviceCount{
			{DeviceType: "desktop", Count: 70},
			{DeviceType: "mobile", Count: 50},
		},
		TopReferrers: []ReferrerCount{
			{Referrer: "facebook.com", Count: 90},
			{Referrer: "direct", Count: 30},
		},
		ClicksOverTime: []TimeCount{
			{Timestamp: dateFrom, Count: 20},
			{Timestamp: dateFrom.Add(24 * time.Hour), Count: 30},
			{Timestamp: dateFrom.Add(48 * time.Hour), Count: 25},
			{Timestamp: dateTo, Count: 45},
		},
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, dataset); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	html := buf.String()

	// Assertions to verify rendering is correct
	if !strings.Contains(html, "TrackFlow") {
		t.Error("expected html to contain branding 'TrackFlow'")
	}
	if !strings.Contains(html, "Test Client Inc.") {
		t.Error("expected html to contain client name 'Test Client Inc.'")
	}
	if !strings.Contains(html, "test-report-uuid-12345") {
		t.Error("expected html to contain report ID")
	}
	if !strings.Contains(html, "Spring Sale") {
		t.Error("expected html to contain campaign name 'Spring Sale'")
	}
	if !strings.Contains(html, "abcde") {
		t.Error("expected html to contain short code 'abcde'")
	}
	if !strings.Contains(html, "PL") || !strings.Contains(html, "US") {
		t.Error("expected html to contain country codes")
	}
	if !strings.Contains(html, "svg viewBox") {
		t.Error("expected html to contain SVG chart tag")
	}
	if !strings.Contains(html, "barGradient") {
		t.Error("expected html to contain SVG gradient definitions")
	}
}
