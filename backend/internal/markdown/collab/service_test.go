package collab

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/user"
)

func TestUserWithoutPermissionCannotJoin(t *testing.T) {
	fixture := newTestService(100)
	_, err := fixture.service.Join(context.Background(), testUser("other-1", "Other"), "doc-1")

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestViewerCanJoinButCannotSubmitUpdate(t *testing.T) {
	fixture := newTestService(100)
	fixture.permissions.view["doc-1:viewer-1"] = true
	session, err := fixture.service.Join(context.Background(), testUser("viewer-1", "Viewer"), "doc-1")
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	defer fixture.service.Leave(session)

	err = fixture.service.ApplyClientMessage(context.Background(), session, ClientUpdate{Type: "update", Content: "blocked"})

	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if fixture.repo.updateCount() != 0 {
		t.Fatalf("expected no persisted updates, got %d", fixture.repo.updateCount())
	}
}

func TestPlainTextDocumentCannotJoin(t *testing.T) {
	fixture := newTestService(100)
	fixture.docs.doc.FileExt = "txt"
	fixture.permissions.edit["doc-1:editor-1"] = true

	_, err := fixture.service.Join(context.Background(), testUser("editor-1", "Editor"), "doc-1")
	if !errors.Is(err, ErrUnsupportedFile) {
		t.Fatalf("expected ErrUnsupportedFile, got %v", err)
	}
}

func TestEditorUpdatePersistsAndBroadcastsToOtherClients(t *testing.T) {
	fixture := newTestService(100)
	fixture.permissions.edit["doc-1:editor-1"] = true
	fixture.permissions.view["doc-1:viewer-1"] = true
	editor, err := fixture.service.Join(context.Background(), testUser("editor-1", "Editor"), "doc-1")
	if err != nil {
		t.Fatalf("join editor: %v", err)
	}
	defer fixture.service.Leave(editor)
	viewer, err := fixture.service.Join(context.Background(), testUser("viewer-1", "Viewer"), "doc-1")
	if err != nil {
		t.Fatalf("join viewer: %v", err)
	}
	defer fixture.service.Leave(viewer)
	drainMessages(t, viewer.Receive)

	if err := fixture.service.ApplyClientMessage(context.Background(), editor, ClientUpdate{Type: "update", Content: "changed"}); err != nil {
		t.Fatalf("apply update: %v", err)
	}

	message := receiveMessage(t, viewer.Receive)
	if message.Type != "update" || message.Content != "changed" {
		t.Fatalf("expected update broadcast, got %#v", message)
	}
	if fixture.repo.updateCount() != 1 {
		t.Fatalf("expected persisted update, got %d", fixture.repo.updateCount())
	}
}

func TestSnapshotRecoveryReplaysUpdates(t *testing.T) {
	fixture := newTestService(100)
	createdBy := "editor-1"
	fixture.repo.snapshots = append(fixture.repo.snapshots, Snapshot{
		ID:         "snapshot-1",
		DocumentID: "doc-1",
		VersionNo:  2,
		Content:    "snapshot",
		CreatedBy:  &createdBy,
		CreatedAt:  time.Now(),
	})
	fixture.repo.updates = append(fixture.repo.updates, Update{
		ID:         "update-3",
		DocumentID: "doc-1",
		UpdateSeq:  3,
		UpdateData: []byte(`{"type":"update","content":"latest"}`),
		CreatedBy:  &createdBy,
		CreatedAt:  time.Now(),
	})
	fixture.permissions.edit["doc-1:editor-1"] = true

	snapshot, err := fixture.service.Snapshot(context.Background(), testUser("editor-1", "Editor"), "doc-1")

	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Content != "latest" || snapshot.VersionNo != 3 {
		t.Fatalf("expected replayed latest content, got %#v", snapshot)
	}
}

func TestSnapshotCreatedAtInterval(t *testing.T) {
	fixture := newTestService(2)
	fixture.permissions.edit["doc-1:editor-1"] = true
	editor, err := fixture.service.Join(context.Background(), testUser("editor-1", "Editor"), "doc-1")
	if err != nil {
		t.Fatalf("join editor: %v", err)
	}
	defer fixture.service.Leave(editor)

	if err := fixture.service.ApplyClientMessage(context.Background(), editor, ClientUpdate{Type: "update", Content: "one"}); err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := fixture.service.ApplyClientMessage(context.Background(), editor, ClientUpdate{Type: "update", Content: "two"}); err != nil {
		t.Fatalf("second update: %v", err)
	}

	if len(fixture.repo.snapshots) != 1 {
		t.Fatalf("expected one generated snapshot, got %d", len(fixture.repo.snapshots))
	}
	if fixture.repo.snapshots[0].Content != "two" || fixture.repo.snapshots[0].VersionNo != 2 {
		t.Fatalf("unexpected snapshot: %#v", fixture.repo.snapshots[0])
	}
}

type testFixture struct {
	service     *Service
	repo        *memoryCollabRepo
	docs        *memoryDocumentRepo
	permissions *memoryPermissionService
}

func newTestService(snapshotInterval int) testFixture {
	repo := &memoryCollabRepo{}
	docs := &memoryDocumentRepo{
		doc: document.Document{
			ID:         "doc-1",
			OwnerID:    "owner-1",
			FileExt:    "md",
			StorageKey: "documents/doc-1/versions/one/example.md",
		},
	}
	objectStorage := &memoryObjectStorage{objects: map[string][]byte{
		"documents/doc-1/versions/one/example.md": []byte("initial"),
	}}
	permissions := &memoryPermissionService{
		view: map[string]bool{},
		edit: map[string]bool{},
	}
	return testFixture{
		service:     NewService(repo, docs, permissions, objectStorage, snapshotInterval),
		repo:        repo,
		docs:        docs,
		permissions: permissions,
	}
}

func testUser(id string, displayName string) user.User {
	return user.User{ID: id, DisplayName: displayName, Email: id + "@example.test"}
}

func receiveMessage(t *testing.T, ch <-chan ServerMessage) ServerMessage {
	t.Helper()
	select {
	case message := <-ch:
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
	return ServerMessage{}
}

func drainMessages(t *testing.T, ch <-chan ServerMessage) {
	t.Helper()
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

type memoryCollabRepo struct {
	snapshots []Snapshot
	updates   []Update
}

func (r *memoryCollabRepo) LatestSnapshot(_ context.Context, documentID string) (Snapshot, error) {
	for i := len(r.snapshots) - 1; i >= 0; i-- {
		if r.snapshots[i].DocumentID == documentID {
			return r.snapshots[i], nil
		}
	}
	return Snapshot{}, ErrNotFound
}

func (r *memoryCollabRepo) ListUpdatesAfter(_ context.Context, documentID string, updateSeq int64) ([]Update, error) {
	var updates []Update
	for _, update := range r.updates {
		if update.DocumentID == documentID && update.UpdateSeq > updateSeq {
			updates = append(updates, update)
		}
	}
	return updates, nil
}

func (r *memoryCollabRepo) NextUpdateSeq(_ context.Context, documentID string) (int64, error) {
	next := int64(1)
	for _, update := range r.updates {
		if update.DocumentID == documentID && update.UpdateSeq >= next {
			next = update.UpdateSeq + 1
		}
	}
	return next, nil
}

func (r *memoryCollabRepo) CreateUpdate(_ context.Context, update Update) error {
	r.updates = append(r.updates, update)
	return nil
}

func (r *memoryCollabRepo) CreateSnapshot(_ context.Context, snapshot Snapshot) error {
	r.snapshots = append(r.snapshots, snapshot)
	return nil
}

func (r *memoryCollabRepo) updateCount() int {
	return len(r.updates)
}

type memoryDocumentRepo struct {
	doc document.Document
}

func (r *memoryDocumentRepo) FindByID(_ context.Context, id string) (document.Document, error) {
	if r.doc.ID != id {
		return document.Document{}, document.ErrNotFound
	}
	return r.doc, nil
}

type memoryPermissionService struct {
	view map[string]bool
	edit map[string]bool
}

func (s *memoryPermissionService) CanView(_ context.Context, userID string, documentID string) (bool, error) {
	key := documentID + ":" + userID
	return s.view[key] || s.edit[key], nil
}

func (s *memoryPermissionService) CanEdit(_ context.Context, userID string, documentID string) (bool, error) {
	return s.edit[documentID+":"+userID], nil
}

type memoryObjectStorage struct {
	objects map[string][]byte
}

func (s *memoryObjectStorage) PutObject(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	s.objects[key] = data
	return nil
}

func (s *memoryObjectStorage) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, document.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *memoryObjectStorage) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func (s *memoryObjectStorage) PresignedGetURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
