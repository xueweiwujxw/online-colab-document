package office

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/user"
)

func TestSessionAllowsViewerAndReturnsReadOnlyMode(t *testing.T) {
	h := newHarness()
	h.permissions.canView = true
	h.permissions.canEdit = false

	session, err := h.service.Session(context.Background(), testUser(), "doc-1")
	if err != nil {
		t.Fatalf("Session returned error: %v", err)
	}
	if session.Mode != "view" {
		t.Fatalf("mode = %q, want view", session.Mode)
	}
	if session.Provider != "casual" {
		t.Fatalf("provider = %q, want casual", session.Provider)
	}
	if session.DownloadURL != "http://api.example.test/api/documents/doc-1/download" {
		t.Fatalf("download url = %q", session.DownloadURL)
	}
	if session.EditorURL == "" {
		t.Fatal("editor URL is empty")
	}
}

func TestSessionRejectsUnsupportedFile(t *testing.T) {
	h := newHarness()
	h.repo.doc.FileExt = "doc"

	_, err := h.service.Session(context.Background(), testUser(), "doc-1")
	if err != ErrUnsupportedFile {
		t.Fatalf("error = %v, want ErrUnsupportedFile", err)
	}
}

func TestCollabSessionReturnsWriteRoleForEditor(t *testing.T) {
	h := newHarness()
	h.repo.doc.FileExt = "xlsx"
	h.permissions.canView = true
	h.permissions.canEdit = true
	h.service.cfg.CollabEnabled = true
	h.service.cfg.CollabPublicURL = "ws://collab.example.test/yjs/"

	session, err := h.service.CollabSession(context.Background(), testUser(), "doc-1")
	if err != nil {
		t.Fatalf("CollabSession returned error: %v", err)
	}
	if !session.Enabled {
		t.Fatalf("enabled = false, want true")
	}
	if session.Role != "write" {
		t.Fatalf("role = %q, want write", session.Role)
	}
	if session.Room != "office:doc-1" {
		t.Fatalf("room = %q, want office:doc-1", session.Room)
	}
	if session.ServerURL != "ws://collab.example.test/yjs" {
		t.Fatalf("server url = %q", session.ServerURL)
	}
}

func TestCollabSessionReturnsViewRoleForViewer(t *testing.T) {
	h := newHarness()
	h.repo.doc.FileExt = "xlsx"
	h.permissions.canView = true
	h.permissions.canEdit = false
	h.service.cfg.CollabEnabled = true
	h.service.cfg.CollabPublicURL = "ws://collab.example.test/yjs"

	session, err := h.service.CollabSession(context.Background(), testUser(), "doc-1")
	if err != nil {
		t.Fatalf("CollabSession returned error: %v", err)
	}
	if session.Role != "view" {
		t.Fatalf("role = %q, want view", session.Role)
	}
}

func TestCollabSessionDisabledWhenServiceNotConfigured(t *testing.T) {
	h := newHarness()
	h.repo.doc.FileExt = "xlsx"

	session, err := h.service.CollabSession(context.Background(), testUser(), "doc-1")
	if err != nil {
		t.Fatalf("CollabSession returned error: %v", err)
	}
	if session.Enabled {
		t.Fatalf("enabled = true, want false")
	}
	if session.Reason == "" {
		t.Fatalf("reason is empty")
	}
}

func TestCollabSessionRejectsDocx(t *testing.T) {
	h := newHarness()
	h.repo.doc.FileExt = "docx"

	_, err := h.service.CollabSession(context.Background(), testUser(), "doc-1")
	if err != ErrUnsupportedFile {
		t.Fatalf("error = %v, want ErrUnsupportedFile", err)
	}
}

func TestSaveRequiresEditPermission(t *testing.T) {
	h := newHarness()
	h.permissions.canView = true
	h.permissions.canEdit = false

	_, _, err := h.service.Save(context.Background(), testUser(), "doc-1", bytes.NewReader([]byte("updated")))
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestSaveStoresNewVersion(t *testing.T) {
	h := newHarness()
	h.permissions.canView = true
	h.permissions.canEdit = true

	doc, size, err := h.service.Save(context.Background(), testUser(), "doc-1", bytes.NewReader([]byte("updated")))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if doc.ID != "doc-1" {
		t.Fatalf("doc id = %q, want doc-1", doc.ID)
	}
	if size != 7 {
		t.Fatalf("size = %d, want 7", size)
	}
	if h.storage.objects["documents/doc-1/versions/version-2/example.docx"] != "updated" {
		t.Fatalf("stored object = %q", h.storage.objects["documents/doc-1/versions/version-2/example.docx"])
	}
	if h.repo.addedVersion.ID != "version-2" {
		t.Fatalf("version id = %q, want version-2", h.repo.addedVersion.ID)
	}
	if h.repo.addedVersion.CreatedBy == nil || *h.repo.addedVersion.CreatedBy != "user-1" {
		t.Fatalf("created by = %v, want user-1", h.repo.addedVersion.CreatedBy)
	}
}

func TestWOPIRechecksPermissionAfterTokenWasIssued(t *testing.T) {
	h := newHarness()
	token, err := h.service.mintWOPIToken(testUser(), "doc-1", "editor", "docs")
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	h.permissions.canEdit = false
	_, _, err = h.service.WOPISave(context.Background(), token, "doc-1", bytes.NewReader([]byte("blocked")))
	if err != ErrForbidden {
		t.Fatalf("WOPISave error = %v, want ErrForbidden", err)
	}
}

func TestWOPIRejectsTokenForAnotherDocument(t *testing.T) {
	h := newHarness()
	token, err := h.service.mintWOPIToken(testUser(), "doc-1", "editor", "docs")
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	_, _, _, err = h.service.WOPIInfo(context.Background(), token, "another-doc")
	if err != ErrForbidden {
		t.Fatalf("WOPIInfo error = %v, want ErrForbidden", err)
	}
}

type harness struct {
	repo        *fakeRepo
	permissions *fakePermissions
	storage     *fakeStorage
	service     *Service
}

func newHarness() harness {
	repo := &fakeRepo{doc: document.Document{
		ID:               "doc-1",
		OriginalFilename: "example.docx",
		FileExt:          "docx",
		MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		StorageKey:       "documents/doc-1/versions/version-1/example.docx",
	}}
	permissions := &fakePermissions{canView: true, canEdit: true}
	storage := &fakeStorage{objects: map[string]string{}}
	service := NewService(
		Config{
			Provider:        "casual",
			PublicAPIURL:    "http://api.example.test",
			DocsEditorURL:   "http://docs.example.test",
			SheetsEditorURL: "http://sheets.example.test",
			JWTSecret:       "test-casual-secret-at-least-16",
			MaxUploadBytes:  1024,
		},
		repo,
		permissions,
		storage,
	)
	service.newID = func() (string, error) { return "version-2", nil }
	service.now = func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) }
	return harness{repo: repo, permissions: permissions, storage: storage, service: service}
}

func testUser() user.User {
	return user.User{ID: "user-1", DisplayName: "User One"}
}

type fakeRepo struct {
	doc          document.Document
	addedVersion document.Version
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (document.Document, error) {
	if id != r.doc.ID {
		return document.Document{}, document.ErrNotFound
	}
	return r.doc, nil
}

func (r *fakeRepo) AddDocumentVersion(_ context.Context, _ string, version document.Version, _ time.Time) error {
	r.addedVersion = version
	return nil
}

type fakePermissions struct {
	canView bool
	canEdit bool
}

func (p *fakePermissions) CanView(context.Context, string, string) (bool, error) {
	return p.canView, nil
}

func (p *fakePermissions) CanEdit(context.Context, string, string) (bool, error) {
	return p.canEdit, nil
}

type fakeStorage struct {
	objects map[string]string
}

func (s *fakeStorage) PutObject(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.objects[key] = string(data)
	return nil
}

func (s *fakeStorage) GetObject(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}

func (s *fakeStorage) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *fakeStorage) PresignedGetURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
