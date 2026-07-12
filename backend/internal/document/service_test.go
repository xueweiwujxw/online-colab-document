package document

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestUploadSuccessCreatesDocumentVersionAndObject(t *testing.T) {
	repo := newMemoryRepo()
	objectStorage := newMemoryStorage()
	service := NewService(repo, objectStorage, nil, 1024)

	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        int64(len("hello")),
		Reader:           strings.NewReader("hello"),
	})

	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if doc.FileExt != "md" || doc.Title != "example" {
		t.Fatalf("unexpected document metadata: %#v", doc)
	}
	if repo.documentCount() != 1 || repo.versionCount() != 1 {
		t.Fatalf("expected document and version records, got documents=%d versions=%d", repo.documentCount(), repo.versionCount())
	}
	if _, ok := objectStorage.objects[doc.StorageKey]; !ok {
		t.Fatalf("expected storage object at %q", doc.StorageKey)
	}
}

func TestUploadAcceptsRequiredFileTypes(t *testing.T) {
	cases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{name: "docx", filename: "example.docx", contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{name: "xlsx", filename: "example.xlsx", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{name: "markdown", filename: "example.md", contentType: "text/markdown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewService(newMemoryRepo(), newMemoryStorage(), nil, 1024)
			_, err := service.Upload(context.Background(), UploadInput{
				OwnerID:          "owner-1",
				OriginalFilename: tc.filename,
				HeaderMimeType:   tc.contentType,
				SizeBytes:        5,
				Reader:           strings.NewReader("hello"),
			})
			if err != nil {
				t.Fatalf("upload %s: %v", tc.filename, err)
			}
		})
	}
}

func TestUploadRequiresAuthenticatedUser(t *testing.T) {
	handler := NewHandler(NewService(newMemoryRepo(), newMemoryStorage(), nil, 1024), slog.Default())
	body, contentType := multipartBody(t, "file", "example.md", "text/markdown", "hello")
	req := httptest.NewRequest(http.MethodPost, "/api/documents/upload", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handler.Upload(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestUploadUnsupportedExtensionFails(t *testing.T) {
	service := NewService(newMemoryRepo(), newMemoryStorage(), nil, 1024)

	_, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "malware.exe",
		HeaderMimeType:   "application/octet-stream",
		SizeBytes:        4,
		Reader:           strings.NewReader("test"),
	})

	if err != ErrUnsupportedType {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestUploadTooLargeFails(t *testing.T) {
	service := NewService(newMemoryRepo(), newMemoryStorage(), nil, 3)

	_, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        4,
		Reader:           strings.NewReader("test"),
	})

	if err != ErrFileTooLarge {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestDownloadSuccess(t *testing.T) {
	repo := newMemoryRepo()
	objectStorage := newMemoryStorage()
	service := NewService(repo, objectStorage, nil, 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	download, err := service.Download(context.Background(), "owner-1", doc.ID)

	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer download.Reader.Close()
	data, err := io.ReadAll(download.Reader)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected downloaded content hello, got %q", string(data))
	}
}

func TestNonOwnerDownloadFails(t *testing.T) {
	service := NewService(newMemoryRepo(), newMemoryStorage(), nil, 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	_, err = service.Download(context.Background(), "owner-2", doc.ID)

	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestEditorCanDownloadThroughPermissionService(t *testing.T) {
	repo := newMemoryRepo()
	objectStorage := newMemoryStorage()
	permissions := newFakePermissionService()
	service := NewService(repo, objectStorage, permissions, 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	permissions.view[doc.ID+":editor-1"] = true

	download, err := service.Download(context.Background(), "editor-1", doc.ID)

	if err != nil {
		t.Fatalf("download as editor: %v", err)
	}
	_ = download.Reader.Close()
}

func TestViewerCannotDeleteThroughPermissionService(t *testing.T) {
	repo := newMemoryRepo()
	permissions := newFakePermissionService()
	service := NewService(repo, newMemoryStorage(), permissions, 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	permissions.view[doc.ID+":viewer-1"] = true

	err = service.Delete(context.Background(), "viewer-1", doc.ID)

	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUserWithoutPermissionCannotViewDocument(t *testing.T) {
	service := NewService(newMemoryRepo(), newMemoryStorage(), newFakePermissionService(), 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	_, err = service.Get(context.Background(), "other-1", doc.ID)

	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteSuccessAndListHidesDeleted(t *testing.T) {
	service := NewService(newMemoryRepo(), newMemoryStorage(), nil, 1024)
	doc, err := service.Upload(context.Background(), UploadInput{
		OwnerID:          "owner-1",
		OriginalFilename: "example.md",
		HeaderMimeType:   "text/markdown",
		SizeBytes:        5,
		Reader:           strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if err := service.Delete(context.Background(), "owner-1", doc.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	docs, err := service.List(context.Background(), "owner-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected deleted document hidden from list, got %d", len(docs))
	}
}

func multipartBody(t *testing.T, field string, filename string, contentType string, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="` + field + `"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

type memoryRepo struct {
	mu        sync.Mutex
	documents map[string]Document
	versions  map[string][]Version
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		documents: map[string]Document{},
		versions:  map[string][]Version{},
	}
}

func (r *memoryRepo) CreateWithVersion(_ context.Context, doc Document, version Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.documents[doc.ID] = doc
	r.versions[doc.ID] = append(r.versions[doc.ID], version)
	return nil
}

func (r *memoryRepo) ListByOwner(_ context.Context, ownerID string) ([]Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	docs := make([]Document, 0)
	for _, doc := range r.documents {
		if doc.OwnerID == ownerID && doc.DeletedAt == nil {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func (r *memoryRepo) ListAccessible(_ context.Context, userID string) ([]Document, error) {
	return r.ListByOwner(context.Background(), userID)
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.documents[id]
	if !ok || doc.DeletedAt != nil {
		return Document{}, ErrNotFound
	}
	return doc, nil
}

func (r *memoryRepo) SoftDeleteForOwner(_ context.Context, id string, ownerID string, deletedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.documents[id]
	if !ok || doc.OwnerID != ownerID || doc.DeletedAt != nil {
		return ErrNotFound
	}
	doc.DeletedAt = &deletedAt
	doc.UpdatedAt = deletedAt
	r.documents[id] = doc
	return nil
}

func (r *memoryRepo) ListVersions(_ context.Context, documentID string) ([]Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, ok := r.documents[documentID]
	if !ok || doc.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return append([]Version(nil), r.versions[documentID]...), nil
}

func (r *memoryRepo) documentCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.documents)
}

func (r *memoryRepo) versionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, versions := range r.versions {
		total += len(versions)
	}
	return total
}

type memoryStorage struct {
	objects map[string][]byte
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{objects: map[string][]byte{}}
}

func (s *memoryStorage) PutObject(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.objects[key] = data
	return nil
}

func (s *memoryStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *memoryStorage) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *memoryStorage) PresignedGetURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

type fakePermissionService struct {
	view   map[string]bool
	manage map[string]bool
	delete map[string]bool
}

func newFakePermissionService() *fakePermissionService {
	return &fakePermissionService{
		view:   map[string]bool{},
		manage: map[string]bool{},
		delete: map[string]bool{},
	}
}

func (s *fakePermissionService) CanView(_ context.Context, userID string, documentID string) (bool, error) {
	return s.view[documentID+":"+userID], nil
}

func (s *fakePermissionService) CanManage(_ context.Context, userID string, documentID string) (bool, error) {
	return s.manage[documentID+":"+userID], nil
}

func (s *fakePermissionService) CanDelete(_ context.Context, userID string, documentID string) (bool, error) {
	return s.delete[documentID+":"+userID], nil
}
