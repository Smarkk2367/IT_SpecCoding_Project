package stats

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"trackflow/internal/auth"
	"trackflow/internal/httpx"
)

type Handler struct {
	stats *Repository
}

func NewHandler(stats *Repository) Handler {
	return Handler{stats: stats}
}

type response struct {
	TotalClicks    int             `json:"total_clicks"`
	UniqueClicks   int             `json:"unique_clicks"`
	ClicksOverTime []timeCount     `json:"clicks_over_time"`
	ByCountry      []countryCount  `json:"by_country"`
	ByDevice       []deviceCount   `json:"by_device"`
	ByReferrer     []referrerCount `json:"by_referrer"`
}

type timeCount struct {
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}

type countryCount struct {
	Country string `json:"country"`
	Count   int    `json:"count"`
}

type deviceCount struct {
	DeviceType string `json:"device_type"`
	Count      int    `json:"count"`
}

type referrerCount struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || (claims.Role != "marketer" && claims.Role != "client") {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "auth role required")
		return
	}

	query := Query{
		LinkID: chi.URLParam(r, "id"),
		Period: r.URL.Query().Get("period"),
	}
	if value := r.URL.Query().Get("date_from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_from must be ISO8601")
			return
		}
		query.DateFrom = &parsed
	}
	if value := r.URL.Query().Get("date_to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "date_to must be ISO8601")
			return
		}
		query.DateTo = &parsed
	}

	result, err := h.stats.Get(r.Context(), query)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load stats")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toStatsResponse(result))
}

func toStatsResponse(result Result) response {
	out := response{
		TotalClicks:  result.TotalClicks,
		UniqueClicks: result.UniqueClicks,
	}
	for _, row := range result.ClicksOverTime {
		out.ClicksOverTime = append(out.ClicksOverTime, timeCount{
			Timestamp: row.Timestamp.UTC().Format(time.RFC3339),
			Count:     row.Count,
		})
	}
	for _, row := range result.ByCountry {
		out.ByCountry = append(out.ByCountry, countryCount{
			Country: row.Country,
			Count:   row.Count,
		})
	}
	for _, row := range result.ByDevice {
		out.ByDevice = append(out.ByDevice, deviceCount{
			DeviceType: row.DeviceType,
			Count:      row.Count,
		})
	}
	for _, row := range result.ByReferrer {
		out.ByReferrer = append(out.ByReferrer, referrerCount{
			Referrer: row.Referrer,
			Count:    row.Count,
		})
	}
	return out
}
