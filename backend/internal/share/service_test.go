package share

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"online-colab-document/backend/internal/document"
)

func TestOwnerCreatesShareLinkWithHashedToken(t *testing.T) {
	fixture := newTestFixture()
	fixture.permissions.canShare["doc-1:owner-1"] = true
	fixture.service.newToken = func() (string, error) { return "plain-token", nil }

	response, err := fixture.service.Create(context.Background(), CreateInput{
		ActorID:    "owner-1",
		DocumentID: "doc-1",
		Permission: PermissionViewer,
	})

	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if response.Token != "plain-token" {
		t.Fatalf("expected plain token returned once, got %q", response.Token)
	}
	stored := fixture.repo.links[response.ID]
	if stored.TokenHash == "plain-token" || stored.TokenHash != HashToken("plain-token") {
		t.Fatalf("expected stored hash, got %q", stored.TokenHash)
	}
}

func TestEditorCannotCreateShareLink(t *testing.T) {
	fixture := newTestFixture()

	_, err := fixture.service.Create(context.Background(), CreateInput{
		ActorID:    "editor-1",
		DocumentID: "doc-1",
		Permission: PermissionViewer,
	})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestExpiredAndDisabledLinksCannotAccess(t *testing.T) {
	fixture := newTestFixture()
	expired := fixture.now.Add(-time.Minute)
	fixture.repo.links["expired"] = Link{
		ID:         "expired",
		DocumentID: "doc-1",
		TokenHash:  HashToken("expired-token"),
		Permission: PermissionViewer,
		ExpiresAt:  &expired,
		CreatedAt:  fixture.now,
	}
	fixture.repo.links["disabled"] = Link{
		ID:         "disabled",
		DocumentID: "doc-1",
		TokenHash:  HashToken("disabled-token"),
		Permission: PermissionViewer,
		Disabled:   true,
		CreatedAt:  fixture.now,
	}

	if _, err := fixture.service.Access(context.Background(), "expired-token"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
	if _, err := fixture.service.Access(context.Background(), "disabled-token"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestViewerLinkCannotEditAndEditorLinkCanEdit(t *testing.T) {
	fixture := newTestFixture()
	fixture.repo.links["viewer"] = Link{
		ID:         "viewer",
		DocumentID: "doc-1",
		TokenHash:  HashToken("viewer-token"),
		Permission: PermissionViewer,
		CreatedAt:  fixture.now,
	}
	fixture.repo.links["editor"] = Link{
		ID:         "editor",
		DocumentID: "doc-1",
		TokenHash:  HashToken("editor-token"),
		Permission: PermissionEditor,
		CreatedAt:  fixture.now,
	}

	if _, err := fixture.service.SaveMarkdown(context.Background(), "viewer-token", "blocked"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected viewer ErrForbidden, got %v", err)
	}

	response, err := fixture.service.SaveMarkdown(context.Background(), "editor-token", "updated")
	if err != nil {
		t.Fatalf("editor save: %v", err)
	}
	if !response.CanEdit || response.Content != "updated" {
		t.Fatalf("unexpected editor response: %#v", response)
	}
	if fixture.docs.versionCount != 1 {
		t.Fatalf("expected one markdown version, got %d", fixture.docs.versionCount)
	}
	download, err := fixture.service.Download(context.Background(), "editor-token")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer download.Reader.Close()
	data, err := io.ReadAll(download.Reader)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(data) != "updated" {
		t.Fatalf("expected latest content, got %q", string(data))
	}
}

func TestListAndDisableRequireOwnerSharePermission(t *testing.T) {
	fixture := newTestFixture()
	fixture.permissions.canShare["doc-1:owner-1"] = true
	fixture.repo.links["link-1"] = Link{
		ID:         "link-1",
		DocumentID: "doc-1",
		TokenHash:  HashToken("token"),
		Permission: PermissionViewer,
		CreatedAt:  fixture.now,
	}

	if _, err := fixture.service.List(context.Background(), "editor-1", "doc-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected list ErrForbidden, got %v", err)
	}
	links, err := fixture.service.List(context.Background(), "owner-1", "doc-1")
	if err != nil {
		t.Fatalf("list owner: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one link, got %d", len(links))
	}
	if err := fixture.service.Disable(context.Background(), "owner-1", "link-1"); err != nil {
		t.Fatalf("disable owner: %v", err)
	}
	if !fixture.repo.links["link-1"].Disabled {
		t.Fatalf("expected link disabled")
	}
}

type testFixture struct {
	service     *Service
	repo        *memoryRepo
	docs        *memoryDocumentRepo
	permissions *memoryPermissionService
	now         time.Time
}

func newTestFixture() testFixture {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	repo := &memoryRepo{links: map[string]Link{}}
	docs := &memoryDocumentRepo{
		doc: document.Document{
			ID:               "doc-1",
			OwnerID:          "owner-1",
			Title:            "Example",
			OriginalFilename: "example.md",
			FileExt:          "md",
			MimeType:         "text/markdown",
			StorageKey:       "documents/doc-1/versions/one/example.md",
			SizeBytes:        int64(len("hello")),
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	objectStorage := &memoryStorage{objects: map[string][]byte{
		"documents/doc-1/versions/one/example.md": []byte("hello"),
	}}
	permissions := &memoryPermissionService{canShare: map[string]bool{}}
	service := NewService(repo, docs, permissions, objectStorage, "http://localhost:3000")
	service.now = func() time.Time { return now }
	service.newID = sequentialIDs("id-")
	return testFixture{
		service:     service,
		repo:        repo,
		docs:        docs,
		permissions: permissions,
		now:         now,
	}
}

func sequentialIDs(prefix string) func() (string, error) {
	count := 0
	return func() (string, error) {
		count++
		return prefix + strings.Repeat("x", count), nil
	}
}

type memoryRepo struct {
	links map[string]Link
}

func (r *memoryRepo) Create(_ context.Context, link Link) error {
	r.links[link.ID] = link
	return nil
}

func (r *memoryRepo) ListForDocument(_ context.Context, documentID string) ([]Link, error) {
	var links []Link
	for _, link := range r.links {
		if link.DocumentID == documentID {
			links = append(links, link)
		}
	}
	return links, nil
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (Link, error) {
	link, ok := r.links[id]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}

func (r *memoryRepo) FindByTokenHash(_ context.Context, tokenHash string) (Link, error) {
	for _, link := range r.links {
		if link.TokenHash == tokenHash {
			return link, nil
		}
	}
	return Link{}, ErrNotFound
}

func (r *memoryRepo) Disable(_ context.Context, id string) error {
	link, ok := r.links[id]
	if !ok {
		return ErrNotFound
	}
	link.Disabled = true
	r.links[id] = link
	return nil
}

type memoryDocumentRepo struct {
	doc          document.Document
	versionCount int
}

func (r *memoryDocumentRepo) FindByID(_ context.Context, id string) (document.Document, error) {
	if r.doc.ID != id {
		return document.Document{}, document.ErrNotFound
	}
	return r.doc, nil
}

func (r *memoryDocumentRepo) AddDocumentVersion(_ context.Context, _ string, version document.Version, updatedAt time.Time) error {
	r.versionCount++
	r.doc.StorageKey = version.StorageKey
	r.doc.SizeBytes = version.SizeBytes
	r.doc.UpdatedAt = updatedAt
	return nil
}

type memoryPermissionService struct {
	canShare map[string]bool
}

func (s *memoryPermissionService) CanShare(_ context.Context, userID string, documentID string) (bool, error) {
	return s.canShare[documentID+":"+userID], nil
}

type memoryStorage struct {
	objects map[string][]byte
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
		return nil, document.ErrNotFound
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
