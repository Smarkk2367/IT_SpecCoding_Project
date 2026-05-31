package client

import (
	"errors"
	"net/http"
	"time"

	"trackflow/internal/auth"
	"trackflow/internal/httpx"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) Handler {
	return Handler{repo: repo}
}

type clientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type listResponse struct {
	Data []clientResponse `json:"data"`
}

type createRequest struct {
	Name string `json:"name"`
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	clients, err := h.repo.List(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list clients")
		return
	}

	data := make([]clientResponse, 0, len(clients))
	for _, c := range clients {
		data = append(data, clientResponse{
			ID:        c.ID,
			Name:      c.Name,
			CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, listResponse{Data: data})
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "marketer") {
		return
	}

	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	c, err := h.repo.Create(r.Context(), req.Name)
	if errors.Is(err, ErrDuplicate) {
		httpx.WriteError(w, http.StatusConflict, "CLIENT_ALREADY_EXISTS", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create client")
		return
	}

	res := clientResponse{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
	}

	httpx.WriteJSON(w, http.StatusCreated, res)
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
