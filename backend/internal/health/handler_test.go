package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"online-colab-document/backend/internal/config"
)

func TestHealthz(t *testing.T) {
	handler := NewHandler(NewChecker(config.Config{}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestReadyzSkipsUnconfiguredDependencies(t *testing.T) {
	handler := NewHandler(NewChecker(config.Config{}))
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}
	for _, key := range []string{"database", "redis", "storage"} {
		if body.Checks[key] != "skipped" {
			t.Fatalf("expected %s skipped, got %q", key, body.Checks[key])
		}
	}
}

func TestReadyzRejectsStorageWithOpenPortButInvalidCredentials(t *testing.T) {
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code></Error>`))
	}))
	defer storage.Close()
	handler := NewHandler(NewChecker(config.Config{S3Endpoint: storage.URL, S3AccessKey: "test", S3SecretKey: "test-secret", S3Bucket: "docs"}))
	rec := httptest.NewRecorder()
	handler.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("open port must not imply ready: %d", rec.Code)
	}
}
