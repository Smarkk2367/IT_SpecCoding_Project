package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/redis/go-redis/v9"

	"trackflow/internal/event"
	"trackflow/internal/report"
)

type ReportConsumer struct {
	redis          *redis.Client
	reports        *report.Repository
	publisher      event.Publisher
	group          string
	name           string
	pdfStoragePath string
	logger         *slog.Logger
}

func NewReportConsumer(
	redisClient *redis.Client,
	reports *report.Repository,
	publisher event.Publisher,
	group string,
	name string,
	pdfStoragePath string,
	logger *slog.Logger,
) *ReportConsumer {
	return &ReportConsumer{
		redis:          redisClient,
		reports:        reports,
		publisher:      publisher,
		group:          group,
		name:           name,
		pdfStoragePath: pdfStoragePath,
		logger:         logger,
	}
}

func (c *ReportConsumer) Run(ctx context.Context) error {
	if err := c.redis.XGroupCreateMkStream(ctx, event.ReportsStream, c.group, "0").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create report consumer group: %w", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.name,
			Streams:  []string{event.ReportsStream, ">"},
			Count:    5,
			Block:    time.Second,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			c.logger.Warn("read report stream failed", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				if err := c.processMessage(ctx, message); err != nil {
					c.logger.Error("process report message failed", "message_id", message.ID, "error", err)
					// Note: DLQ / failed_events table handling would go here for retry exhaust
				}
				if err := c.redis.XAck(ctx, event.ReportsStream, c.group, message.ID).Err(); err != nil {
					c.logger.Warn("ack report message failed", "message_id", message.ID, "error", err)
				}
			}
		}
	}
}

func (c *ReportConsumer) processMessage(ctx context.Context, message redis.XMessage) error {
	raw, ok := message.Values["event"].(string)
	if !ok || raw == "" {
		return errors.New("missing event field")
	}

	var envelope event.Envelope[event.ReportRequestedPayload]
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return fmt.Errorf("decode report envelope: %w", err)
	}
	if envelope.EventType != event.ReportRequestedType || envelope.Version != event.Version {
		return fmt.Errorf("unsupported event %s version %s", envelope.EventType, envelope.Version)
	}

	reportID := envelope.Payload.ReportID
	c.logger.Info("processing report request", "report_id", reportID)

	// Step 1: Update status to 'processing'
	if err := c.reports.UpdateStatus(ctx, reportID, "processing", nil, nil); err != nil {
		return fmt.Errorf("update report status to processing: %w", err)
	}

	// Helper function to mark report as failed in database
	failReport := func(reason string) error {
		c.logger.Error("report generation failed", "report_id", reportID, "error", reason)
		if dbErr := c.reports.UpdateStatus(ctx, reportID, "failed", nil, &reason); dbErr != nil {
			c.logger.Error("failed to mark report as failed in db", "report_id", reportID, "error", dbErr)
		}
		return errors.New(reason)
	}

	// Step 2: Query report dataset
	dataset, err := c.reports.GetReportDataset(ctx, reportID)
	if err != nil {
		return failReport(fmt.Sprintf("get report dataset: %v", err))
	}

	// Step 3: Build HTML content
	tmpl, err := report.GetTemplate()
	if err != nil {
		return failReport(fmt.Sprintf("get HTML template: %v", err))
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, dataset); err != nil {
		return failReport(fmt.Sprintf("execute template: %v", err))
	}

	// Step 4: Render to PDF using chromedp
	if err := os.MkdirAll(c.pdfStoragePath, 0755); err != nil {
		return failReport(fmt.Sprintf("create storage directory: %v", err))
	}

	filePath := filepath.Join(c.pdfStoragePath, fmt.Sprintf("report_%s.pdf", reportID))

	pdfBytes, err := renderHTMLToPDF(ctx, htmlBuf.String())
	if err != nil {
		return failReport(fmt.Sprintf("render HTML to PDF: %v", err))
	}

	// Step 5: Save PDF to disk
	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return failReport(fmt.Sprintf("save PDF file: %v", err))
	}

	// Step 6: Update status to 'done'
	if err := c.reports.UpdateStatus(ctx, reportID, "done", &filePath, nil); err != nil {
		return fmt.Errorf("update report status to done: %w", err)
	}

	// Query recipient email (the marketer who requested the report)
	var recipientEmail string
	rep, err := c.reports.FindByID(ctx, reportID)
	if err == nil {
		email, emailErr := c.reports.FindUserEmail(ctx, rep.RequestedBy)
		if emailErr == nil {
			recipientEmail = email
		}
	}

	if recipientEmail == "" {
		recipientEmail = "noreply@trackflow.io" // fallback
	}

	// Step 7: Publish notification.send event
	msg := fmt.Sprintf("Twój raport TrackFlow (ID: %s) za okres %s - %s został wygenerowany i jest gotowy do pobrania.",
		reportID,
		dataset.DateFrom.Format("02.01.2006"),
		dataset.DateTo.Format("02.01.2006"),
	)
	notificationPayload := event.NotificationSendPayload{
		Type:           "report_ready",
		RecipientEmail: recipientEmail,
		Subject:        "Twój raport TrackFlow jest gotowy",
		TemplateData: event.NotificationTemplateData{
			ReportID: &reportID,
			Message:  &msg,
		},
	}

	if err := c.publisher.PublishNotificationSend(ctx, notificationPayload); err != nil {
		c.logger.Warn("failed to publish notification event", "report_id", reportID, "error", err)
	}

	c.logger.Info("report generation completed successfully", "report_id", reportID, "file_path", filePath)
	return nil
}

func renderHTMLToPDF(ctx context.Context, html string) ([]byte, error) {
	// headless Chrome settings for Docker container execution
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// We set a 30 second timeout for the entire PDF rendering process
	renderCtx, renderCancel := context.WithTimeout(ctx, 30*time.Second)
	defer renderCancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(renderCtx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	var pdfBuffer []byte
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate("data:text/html;charset=utf-8,"+url.PathEscape(html)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuffer = buf
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	return pdfBuffer, nil
}
