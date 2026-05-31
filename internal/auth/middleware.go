package auth

import (
	"net/http"
	"strings"

	"trackflow/internal/httpx"
)

func Middleware(tokens TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
				return
			}

			claims, err := tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid bearer token")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}
