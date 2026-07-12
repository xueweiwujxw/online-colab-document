package onlyoffice

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/user"
)

func TestConfigNoPermissionFails(t *testing.T) {
	harness := newTestHarness()
	harness.permissions.view = false

	_, err := harness.service.Config(context.Background(), user.User{ID: "viewer-1"}, "doc-1")

	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestViewerGetsViewConfig(t *testing.T) {
	harness := newTestHarness()
	harness.permissions.view = true
	harness.permissions.edit = false

	cfg, err := harness.service.Config(context.Background(), user.User{ID: "viewer-1", DisplayName: "Viewer"}, "doc-1")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.EditorConfig.Mode != "view" {
		t.Fatalf("expected view mode, got %q", cfg.EditorConfig.Mode)
	}
	if cfg.DocumentType != "word" || cfg.Document.URL == "" || cfg.Token == "" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestEditorGetsEditConfig(t *testing.T) {
	harness := newTestHarness()
	harness.permissions.view = true
	harness.permissions.edit = true

	cfg, err := harness.service.Config(context.Background(), user.User{ID: "editor-1", DisplayName: "Editor"}, "doc-1")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.EditorConfig.Mode != "edit" {
		t.Fatalf("expected edit mode, got %q", cfg.EditorConfig.Mode)
	}
}

func TestXLSXGetsCellConfig(t *testing.T) {
	harness := newTestHarness()
	doc := harness.documents.docs["doc-1"]
	doc.OriginalFilename = "sheet.xlsx"
	doc.FileExt = "xlsx"
	doc.MimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	harness.documents.docs["doc-1"] = doc

	cfg, err := harness.service.Config(context.Background(), user.User{ID: "editor-1", DisplayName: "Editor"}, "doc-1")

	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if cfg.DocumentType != "cell" {
		t.Fatalf("expected cell document type, got %q", cfg.DocumentType)
	}
}

func TestConfigRejectsNonOfficeDocument(t *testing.T) {
	harness := newTestHarness()
	doc := harness.documents.docs["doc-1"]
	doc.FileExt = "md"
	harness.documents.docs["doc-1"] = doc

	_, err := harness.service.Config(context.Background(), user.User{ID: "owner-1"}, "doc-1")

	if err != ErrUnsupportedFile {
		t.Fatalf("expected ErrUnsupportedFile, got %v", err)
	}
}

func TestCallbackInvalidTokenFails(t *testing.T) {
	harness := newTestHarness()

	_, err := harness.service.Callback(context.Background(), "doc-1", CallbackRequest{Key: "doc-doc-1-version-1", Status: 2, URL: "http://example.test/file"}, "Bearer invalid")

	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestCallbackSaveCreatesNewVersion(t *testing.T) {
	harness := newTestHarness()
	harness.service.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("new-version"))),
			Header:     make(http.Header),
		}, nil
	})}
	callback := CallbackRequest{Key: "doc-doc-1-version-1", Status: 2, URL: "http://onlyoffice-download.example.test/file"}
	token, err := signJWT(callback, harness.service.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("sign callback: %v", err)
	}
	callback.Token = token

	resp, err := harness.service.Callback(context.Background(), "doc-1", callback, "")

	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if resp.Error != 0 {
		t.Fatalf("expected error 0, got %d", resp.Error)
	}
	if len(harness.documents.versions["doc-1"]) != 1 {
		t.Fatalf("expected one new version, got %d", len(harness.documents.versions["doc-1"]))
	}
	doc := harness.documents.docs["doc-1"]
	if doc.CurrentVersionID == nil || *doc.CurrentVersionID == "version-1" {
		t.Fatalf("expected current version to update, got %#v", doc.CurrentVersionID)
	}
	if string(harness.storage.objects[doc.StorageKey]) != "new-version" {
		t.Fatalf("expected stored new version")
	}
}

type testHarness struct {
	service     *Service
	documents   *fakeDocumentRepo
	permissions *fakePermissionService
	storage     *fakeStorage
}

func newTestHarness() testHarness {
	versionID := "version-1"
	repo := &fakeDocumentRepo{
		docs: map[string]document.Document{
			"doc-1": {
				ID:               "doc-1",
				OwnerID:          "owner-1",
				Title:            "Example",
				OriginalFilename: "example.docx",
				FileExt:          "docx",
				MimeType:         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
				StorageKey:       "documents/doc-1/versions/version-1/example.docx",
				CurrentVersionID: &versionID,
				SizeBytes:        10,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
		},
		versions: map[string][]document.Version{},
		saves:    map[string]bool{},
	}
	permissions := &fakePermissionService{view: true, edit: true}
	storage := &fakeStorage{objects: map[string][]byte{}}
	service := NewService(Config{
		Enabled:          true,
		PublicURL:        "http://onlyoffice.example.test",
		JWTSecret:        "test-secret",
		PublicAPIURL:     "http://api.example.test",
		CallbackBaseURL:  "http://backend.example.test",
		MaxDownloadBytes: 1024,
	}, repo, permissions, storage)
	return testHarness{
		service:     service,
		documents:   repo,
		permissions: permissions,
		storage:     storage,
	}
}

type fakeDocumentRepo struct {
	docs     map[string]document.Document
	versions map[string][]document.Version
	saves    map[string]bool
}

func (r *fakeDocumentRepo) FindByID(_ context.Context, id string) (document.Document, error) {
	doc, ok := r.docs[id]
	if !ok {
		return document.Document{}, document.ErrNotFound
	}
	return doc, nil
}

func (r *fakeDocumentRepo) HasOnlyOfficeSave(_ context.Context, documentID string, documentKey string) (bool, error) {
	return r.saves[documentID+":"+documentKey], nil
}

func (r *fakeDocumentRepo) AddVersion(_ context.Context, documentID string, version document.Version, documentKey string, updatedAt time.Time) (bool, error) {
	key := documentID + ":" + documentKey
	if r.saves[key] {
		return false, nil
	}
	r.saves[key] = true
	version.VersionNo = int64(len(r.versions[documentID]) + 1)
	r.versions[documentID] = append(r.versions[documentID], version)
	doc := r.docs[documentID]
	doc.CurrentVersionID = &version.ID
	doc.StorageKey = version.StorageKey
	doc.SizeBytes = version.SizeBytes
	doc.UpdatedAt = updatedAt
	r.docs[documentID] = doc
	return true, nil
}

type fakePermissionService struct {
	view bool
	edit bool
}

func (s *fakePermissionService) CanView(context.Context, string, string) (bool, error) {
	return s.view, nil
}

func (s *fakePermissionService) CanEdit(context.Context, string, string) (bool, error) {
	return s.edit, nil
}

type fakeStorage struct {
	objects map[string][]byte
}

func (s *fakeStorage) PutObject(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.objects[key] = data
	return nil
}

func (s *fakeStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.objects[key])), nil
}

func (s *fakeStorage) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *fakeStorage) PresignedGetURL(context.Context, string, time.Duration) (string, error) {
	return "http://storage.example.test/file", nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
