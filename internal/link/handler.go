package link

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/auth"
	"trackflow/internal/httpx"
)

type Cache interface {
	Set(ctx context.Context, target Link) error
	Delete(ctx context.Context, shortCode string) error
}

type Handler struct {
	links RepositoryAPI
	cache Cache
}

type RepositoryAPI interface {
	List(ctx context.Context, filter ListFilter) (ListResult, error)
	Create(ctx context.Context, input CreateInput) (Link, error)
	FindByID(ctx context.Context, id string) (Link, error)
	Delete(ctx context.Context, id string) (Link, error)
}

func NewHandler(links RepositoryAPI, cache Cache) Handler {
	return Handler{links: links, cache: cache}
}

type createRequest struct {
	OriginalURL  string  `json:"original_url"`
	CampaignName *string `json:"campaign_name"`
	ClientID     *string `json:"client_id"`
	ExpiresAt    *string `json:"expires_at"`
}

type linkResponse struct {
	ID           string  `json:"id"`
	ShortCode    string  `json:"short_code"`
	OriginalURL  string  `json:"original_url"`
	CampaignName *string `json:"campaign_name"`
	ClientID     *string `json:"client_id"`
	ExpiresAt    *string `json:"expires_at"`
	CreatedAt    string  `json:"created_at"`
}

type listResponse struct {
	Data  []linkResponse `json:"data"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	filter := ListFilter{
		Page:  queryInt(r, "page", 1),
		Limit: queryInt(r, "limit", 20),
	}
	if value := r.URL.Query().Get("client_id"); value != "" {
		filter.ClientID = &value
	}
	if value := r.URL.Query().Get("campaign_name"); value != "" {
		filter.CampaignName = &value
	}
	if value := r.URL.Query().Get("active"); value != "" {
		active, err := strconv.ParseBool(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "active must be boolean")
			return
		}
		filter.Active = &active
	}

	result, err := h.links.List(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list links")
		return
	}

	data := make([]linkResponse, 0, len(result.Data))
	for _, link := range result.Data {
		data = append(data, toResponse(link))
	}

	httpx.WriteJSON(w, http.StatusOK, listResponse{Data: data, Total: result.Total, Page: result.Page})
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
	if req.OriginalURL == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "original_url is required")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "expires_at must be ISO8601")
			return
		}
		expiresAt = &parsed
	}

	created, err := h.links.Create(r.Context(), CreateInput{
		OriginalURL:  req.OriginalURL,
		CampaignName: req.CampaignName,
		ClientID:     req.ClientID,
		ExpiresAt:    expiresAt,
		CreatedBy:    claims.Subject,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create link")
		return
	}
	if h.cache != nil {
		_ = h.cache.Set(r.Context(), created)
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(created))
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	found, err := h.links.FindByID(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "link not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not get link")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(found))
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	deleted, err := h.links.Delete(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete link")
		return
	}
	if h.cache != nil {
		_ = h.cache.Delete(r.Context(), deleted.ShortCode)
	}

	w.WriteHeader(http.StatusNoContent)
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

func toResponse(link Link) linkResponse {
	var expiresAt *string
	if link.ExpiresAt != nil {
		value := link.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &value
	}

	return linkResponse{
		ID:           link.ID,
		ShortCode:    link.ShortCode,
		OriginalURL:  link.OriginalURL,
		CampaignName: link.CampaignName,
		ClientID:     link.ClientID,
		ExpiresAt:    expiresAt,
		CreatedAt:    link.CreatedAt.UTC().Format(time.RFC3339),
	}
}
