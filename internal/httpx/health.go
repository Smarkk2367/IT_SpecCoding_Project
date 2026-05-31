package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type readinessResponse struct {
	Service string            `json:"service"`
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks"`
}

func HealthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Service: service,
			Status:  "ok",
		})
	}
}

func ReadinessHandler(service string, checks map[string]func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 750*time.Millisecond)
		defer cancel()

		status := "ok"
		results := make(map[string]string, len(checks))
		for name, check := range checks {
			if err := check(ctx); err != nil {
				status = "degraded"
				results[name] = err.Error()
				continue
			}
			results[name] = "ok"
		}

		code := http.StatusOK
		if status != "ok" {
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(readinessResponse{
			Service: service,
			Status:  status,
			Checks:  results,
		})
	}
}
