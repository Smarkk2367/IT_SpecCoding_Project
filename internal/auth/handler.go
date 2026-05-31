package auth

import (
	"context"
	"errors"
	"net/http"

	"trackflow/internal/httpx"
	"trackflow/internal/user"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
}

type Handler struct {
	users  UserRepository
	tokens TokenManager
}

func NewHandler(users UserRepository, tokens TokenManager) Handler {
	return Handler{
		users:  users,
		tokens: tokens,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userPayload  `json:"user"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type userPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
		return
	}

	found, err := h.users.FindByEmail(r.Context(), req.Email)
	if errors.Is(err, user.ErrNotFound) || (err == nil && !ComparePassword(found.PasswordHash, req.Password)) {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not authenticate user")
		return
	}

	token, err := h.tokens.Issue(Claims{
		Subject:  found.ID,
		Email:    found.Email,
		Role:     found.Role,
		ClientID: found.ClientID,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not issue token")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User:  toUserPayload(found),
	})
}

func (h Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, userPayload{
		ID:    claims.Subject,
		Email: claims.Email,
		Role:  claims.Role,
	})
}

func (h Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context")
		return
	}

	var req changePasswordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "old_password and new_password are required")
		return
	}

	found, err := h.users.FindByID(r.Context(), claims.Subject)
	if errors.Is(err, user.ErrNotFound) || (err == nil && !ComparePassword(found.PasswordHash, req.OldPassword)) {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load user")
		return
	}

	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not hash password")
		return
	}
	if err := h.users.UpdatePassword(r.Context(), claims.Subject, hash); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toUserPayload(u user.User) userPayload {
	return userPayload{
		ID:    u.ID,
		Email: u.Email,
		Role:  u.Role,
	}
}
