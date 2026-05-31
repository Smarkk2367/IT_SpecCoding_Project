package report

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/auth"
	"trackflow/internal/event"
	"trackflow/internal/httpx"
)

type EventPublisher interface {
	PublishReportRequested(ctx context.Context, payload event.ReportRequestedPayload) error
}

type Handler struct {
	repo            *Repository
	publisher       EventPublisher
	pdfStoragePath  string
}

func NewHandler(repo *Repository, publisher EventPublisher, pdfStoragePath string) Handler {
	return Handler{
		repo:           repo,
		publisher:      publisher,
		pdfStoragePath: pdfStoragePath,
	}
}

type createRequest struct {
	DateFrom  string   `json:"date_from"`
	DateTo    string   `json:"date_to"`
	ClientID  *string  `json:"client_id"`
	LinkIDs   []string `json:"link_ids"`
}

type createResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
}

type reportResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
}

type listResponse struct {
	Data  []reportResponse `json:"data"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
}

type detailResponse struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	DownloadURL  *string `json:"download_url"`
	ErrorMessage *string `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	CompletedAt  *string `json:"completed_at"`
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.Role != "marketer" {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "marketer role required")
		return
	}

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}

	if req.DateFrom == "" || req.DateTo == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_from and date_to are required")
		return
	}

	dateFrom, err := time.Parse(time.RFC3339, req.DateFrom)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_from must be ISO8601")
		return
	}

	dateTo, err := time.Parse(time.RFC3339, req.DateTo)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_to must be ISO8601")
		return
	}

	if !dateFrom.Before(dateTo) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_from must be before date_to")
		return
	}

	created, err := h.repo.Create(r.Context(), CreateInput{
		RequestedBy: claims.Subject,
		ClientID:    req.ClientID,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		LinkIDs:     req.LinkIDs,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create report")
		return
	}

	// We publish the event report.requested
	// Need to query report links to populate the payload accurately if they were generated implicitly from client_id
	finalLinkIDs, _ := h.repo.GetReportLinkIDs(r.Context(), created.ID)

	eventPayload := event.ReportRequestedPayload{
		ReportID:    created.ID,
		RequestedBy: created.RequestedBy,
		DateFrom:    created.DateFrom,
		DateTo:      created.DateTo,
		ClientID:    created.ClientID,
		LinkIDs:     finalLinkIDs,
	}

	if err := h.publisher.PublishReportRequested(r.Context(), eventPayload); err != nil {
		// Log event publish error, but we don't fail HTTP request as the outbox table handles fallback
		// (or we can return HTTP 202 as the DB record is created)
	}

	httpx.WriteJSON(w, http.StatusAccepted, createResponse{
		ReportID: created.ID,
		Status:   created.Status,
	})
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	filter := ListFilter{
		Page:  queryInt(r, "page", 1),
		Limit: queryInt(r, "limit", 20),
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	result, err := h.repo.List(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list reports")
		return
	}

	data := make([]reportResponse, 0, len(result.Data))
	for _, rep := range result.Data {
		var completed *string
		if rep.CompletedAt != nil {
			val := rep.CompletedAt.UTC().Format(time.RFC3339)
			completed = &val
		}
		data = append(data, reportResponse{
			ID:          rep.ID,
			Status:      rep.Status,
			CreatedAt:   rep.CreatedAt.UTC().Format(time.RFC3339),
			CompletedAt: completed,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, listResponse{
		Data:  data,
		Total: result.Total,
		Page:  result.Page,
	})
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	id := chi.URLParam(r, "id")
	rep, err := h.repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "REPORT_NOT_FOUND", "report not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not get report")
		return
	}

	var downloadURL *string
	if rep.Status == "done" {
		val := fmt.Sprintf("/api/reports/%s/download", rep.ID)
		downloadURL = &val
	}

	var completedAt *string
	if rep.CompletedAt != nil {
		val := rep.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &val
	}

	httpx.WriteJSON(w, http.StatusOK, detailResponse{
		ID:           rep.ID,
		Status:       rep.Status,
		DownloadURL:  downloadURL,
		ErrorMessage: rep.ErrorMessage,
		CreatedAt:    rep.CreatedAt.UTC().Format(time.RFC3339),
		CompletedAt:  completedAt,
	})
}

func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	// Require client or marketer
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || (claims.Role != "marketer" && claims.Role != "client") {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return
	}

	id := chi.URLParam(r, "id")
	rep, err := h.repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "REPORT_NOT_FOUND", "report not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not get report")
		return
	}

	if rep.Status != "done" || rep.FilePath == nil || *rep.FilePath == "" {
		httpx.WriteError(w, http.StatusBadRequest, "REPORT_NOT_READY", "report is not completed yet")
		return
	}

	filePath := *rep.FilePath
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		httpx.WriteError(w, http.StatusNotFound, "FILE_NOT_FOUND", "report PDF file was not found on disk")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.pdf", rep.ID))
	http.ServeFile(w, r, filePath)
}

func requireRole(w http.ResponseWriter, r *http.Request, role string) bool {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return false
	}
	if claims.Role != role {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", role+" role required")
		return false
	}
	return true
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
