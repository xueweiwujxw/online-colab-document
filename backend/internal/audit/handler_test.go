package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/user"
)

func TestOrdinaryUserCannotAccessAdminAuditAPI(t *testing.T) {
	handler := NewHandler(NewService(&memoryRepo{}), slog.Default())
	rec := httptest.NewRecorder()
	req := authenticatedAuditRequest(user.User{ID: "user-1", IsAdmin: false})

	handler.List(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestAdminCanAccessAuditAPI(t *testing.T) {
	repo := &memoryRepo{logs: []Log{{
		ID:         "log-1",
		Action:     ActionLogin,
		TargetType: "user",
		TargetID:   "00000000-0000-4000-8000-000000000001",
		Metadata:   map[string]any{"ok": true},
		CreatedAt:  time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
	}}}
	handler := NewHandler(NewService(repo), slog.Default())
	rec := httptest.NewRecorder()
	req := authenticatedAuditRequest(user.User{ID: "admin-1", IsAdmin: true})

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items   []PublicLog `json:"items"`
		HasMore bool        `json:"hasMore"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Action != ActionLogin {
		t.Fatalf("unexpected audit response: %#v", body.Items)
	}
}

func TestAdminAuditListPaginatesAndAcceptsIPFilter(t *testing.T) {
	repo := &memoryRepo{logs: []Log{{ID: "log-1", Action: ActionLogin, TargetType: "user", TargetID: "user-1"}, {ID: "log-2", Action: ActionLogout, TargetType: "user", TargetID: "user-1"}}}
	handler := NewHandler(NewService(repo), slog.Default())
	rec := httptest.NewRecorder()
	req := authenticatedAuditRequest(user.User{ID: "admin-1", IsAdmin: true})
	req.URL.RawQuery = "limit=1&ipAddr=127.0.0.1"

	handler.List(rec, req)

	var body struct {
		Items   []PublicLog `json:"items"`
		HasMore bool        `json:"hasMore"`
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || !body.HasMore {
		t.Fatalf("unexpected paginated audit response: %#v", body)
	}
}

func TestAuditMetadataRemovesSensitiveFields(t *testing.T) {
	repo := &memoryRepo{}
	service := NewService(repo)
	service.newID = func() (string, error) { return "00000000-0000-4000-8000-000000000001", nil }

	if err := service.Record(context.Background(), RecordInput{
		Action:     ActionLogin,
		TargetType: "user",
		TargetID:   "00000000-0000-4000-8000-000000000001",
		Metadata: map[string]any{
			"password": "blocked",
			"apiToken": "blocked",
			"secret":   "blocked",
			"safe":     "kept",
		},
	}); err != nil {
		t.Fatalf("record audit: %v", err)
	}
	metadata := repo.logs[0].Metadata
	if _, ok := metadata["password"]; ok {
		t.Fatalf("metadata contains password")
	}
	if _, ok := metadata["apiToken"]; ok {
		t.Fatalf("metadata contains token")
	}
	if _, ok := metadata["secret"]; ok {
		t.Fatalf("metadata contains secret")
	}
	if metadata["safe"] != "kept" {
		t.Fatalf("expected safe metadata kept, got %#v", metadata)
	}
}

func TestToPublicIncludesActorIdentity(t *testing.T) {
	name, email := "王文", "wangwen@example.test"
	public := ToPublic(Log{ActorDisplayName: &name, ActorEmail: &email})
	if public.ActorDisplayName == nil || *public.ActorDisplayName != name || public.ActorEmail == nil || *public.ActorEmail != email {
		t.Fatalf("unexpected actor identity: %#v", public)
	}
}

func authenticatedAuditRequest(u user.User) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs", nil)
	req.AddCookie(&http.Cookie{Name: "docs_session", Value: "session"})
	rec := httptest.NewRecorder()
	middleware.RequireAuth(auditTestAuthenticator{user: u}, "docs_session", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		req = r
	})).ServeHTTP(rec, req)
	return req
}

type auditTestAuthenticator struct {
	user user.User
}

func (a auditTestAuthenticator) CurrentUser(context.Context, string) (user.User, error) {
	return a.user, nil
}

type memoryRepo struct {
	logs []Log
}

func (r *memoryRepo) Create(_ context.Context, log Log) error {
	r.logs = append(r.logs, log)
	return nil
}

func (r *memoryRepo) List(_ context.Context, _ ListFilter) ([]Log, error) {
	return append([]Log(nil), r.logs...), nil
}
