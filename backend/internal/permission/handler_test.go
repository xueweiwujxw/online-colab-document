package permission

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/user"
)

func TestGrantWritesAudit(t *testing.T) {
	repo := newMemoryRepo()
	repo.owners["doc-1"] = "owner-1"
	handler := NewHandler(NewService(repo), slog.Default())
	recorder := &fakeAuditRecorder{}
	handler = handler.WithAudit(recorder)
	secured := middleware.RequireAuth(
		permissionTestAuthenticator{user: user.User{ID: "owner-1"}},
		"docs_session",
		http.HandlerFunc(handler.Grant),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/documents/doc-1/permissions", strings.NewReader(`{
		"subjectType":"user",
		"subjectId":"viewer-1",
		"permission":"viewer"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "doc-1")
	req.AddCookie(&http.Cookie{Name: "docs_session", Value: "session"})
	rec := httptest.NewRecorder()

	secured.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(recorder.inputs) != 1 {
		t.Fatalf("expected one audit record, got %d", len(recorder.inputs))
	}
	if recorder.inputs[0].Action != audit.ActionPermissionGrant || recorder.inputs[0].TargetID != "doc-1" {
		t.Fatalf("unexpected audit input: %#v", recorder.inputs[0])
	}
}

type permissionTestAuthenticator struct {
	user user.User
}

func (a permissionTestAuthenticator) CurrentUser(context.Context, string) (user.User, error) {
	return a.user, nil
}

type fakeAuditRecorder struct {
	inputs []audit.RecordInput
}

func (r *fakeAuditRecorder) Record(_ context.Context, input audit.RecordInput) error {
	r.inputs = append(r.inputs, input)
	return nil
}
