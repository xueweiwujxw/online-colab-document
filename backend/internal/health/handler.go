package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct {
	checker Checker
}

func NewHandler(checker Checker) Handler {
	return Handler{checker: checker}
}

func (h Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 3*time.Second)
	defer cancel()

	checks := h.checker.Ready(ctx)
	status := "ok"
	code := http.StatusOK
	for _, check := range checks {
		if check == "error" {
			status = "error"
			code = http.StatusServiceUnavailable
			break
		}
	}

	writeJSON(w, code, map[string]any{
		"status": status,
		"checks": checks,
	})
}

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}
