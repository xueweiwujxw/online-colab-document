package collab

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/storage"
	"online-colab-document/backend/internal/user"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrUnsupportedFile = errors.New("unsupported markdown document")
)

type DocumentRepository interface {
	FindByID(ctx context.Context, id string) (document.Document, error)
}

type PermissionService interface {
	CanView(ctx context.Context, userID string, documentID string) (bool, error)
	CanEdit(ctx context.Context, userID string, documentID string) (bool, error)
}

type Service struct {
	repo             Repository
	documents        DocumentRepository
	permissions      PermissionService
	storage          storage.Storage
	snapshotInterval int
	rooms            map[string]*room
	mu               sync.Mutex
	now              func() time.Time
	newID            func() (string, error)
}

type ClientSession struct {
	DocumentID  string
	User        user.User
	CanEdit     bool
	Receive     chan ServerMessage
	service     *Service
	room        *room
	connectedAt time.Time
}

type ClientUpdate struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

type ServerMessage struct {
	Type        string         `json:"type"`
	Content     string         `json:"content,omitempty"`
	CanEdit     bool           `json:"canEdit,omitempty"`
	UpdateSeq   int64          `json:"updateSeq,omitempty"`
	UserID      string         `json:"userId,omitempty"`
	DisplayName string         `json:"displayName,omitempty"`
	Users       []PresenceUser `json:"users,omitempty"`
	Error       string         `json:"error,omitempty"`
}

type room struct {
	documentID string
	content    string
	updateSeq  int64
	clients    map[*ClientSession]struct{}
	mu         sync.Mutex
}

func NewService(repo Repository, documents DocumentRepository, permissions PermissionService, objectStorage storage.Storage, snapshotInterval int) *Service {
	if snapshotInterval <= 0 {
		snapshotInterval = 100
	}
	return &Service{
		repo:             repo,
		documents:        documents,
		permissions:      permissions,
		storage:          objectStorage,
		snapshotInterval: snapshotInterval,
		rooms:            map[string]*room{},
		now:              time.Now,
		newID:            newUUID,
	}
}

func (s *Service) Snapshot(ctx context.Context, currentUser user.User, documentID string) (PublicSnapshot, error) {
	canEdit, err := s.checkAccess(ctx, currentUser.ID, documentID)
	if err != nil {
		return PublicSnapshot{}, err
	}
	content, versionNo, err := s.currentContent(ctx, documentID)
	if err != nil {
		return PublicSnapshot{}, err
	}
	return PublicSnapshot{
		DocumentID: documentID,
		Content:    content,
		VersionNo:  versionNo,
		CanEdit:    canEdit,
		Users:      s.presence(documentID),
	}, nil
}

func (s *Service) Join(ctx context.Context, currentUser user.User, documentID string) (*ClientSession, error) {
	canEdit, err := s.checkAccess(ctx, currentUser.ID, documentID)
	if err != nil {
		return nil, err
	}
	r, err := s.getRoom(ctx, documentID)
	if err != nil {
		return nil, err
	}
	session := &ClientSession{
		DocumentID:  documentID,
		User:        currentUser,
		CanEdit:     canEdit,
		Receive:     make(chan ServerMessage, 16),
		service:     s,
		room:        r,
		connectedAt: s.now().UTC(),
	}
	r.mu.Lock()
	r.clients[session] = struct{}{}
	initial := ServerMessage{
		Type:      "init",
		Content:   r.content,
		CanEdit:   canEdit,
		UpdateSeq: r.updateSeq,
		Users:     r.presenceLocked(),
	}
	r.mu.Unlock()
	session.Receive <- initial
	s.broadcastPresence(r)
	return session, nil
}

func (s *Service) Leave(session *ClientSession) {
	if session == nil || session.room == nil {
		return
	}
	r := session.room
	r.mu.Lock()
	if _, ok := r.clients[session]; ok {
		delete(r.clients, session)
		close(session.Receive)
	}
	r.mu.Unlock()
	s.broadcastPresence(r)
}

func (s *Service) ApplyClientMessage(ctx context.Context, session *ClientSession, update ClientUpdate) error {
	if session == nil {
		return ErrForbidden
	}
	switch update.Type {
	case "presence":
		s.broadcastPresence(session.room)
		return nil
	case "update":
		if !session.CanEdit {
			session.enqueue(ServerMessage{Type: "error", Error: "readonly"})
			return ErrForbidden
		}
		return s.applyContentUpdate(ctx, session, update.Content)
	default:
		return nil
	}
}

func (s *Service) applyContentUpdate(ctx context.Context, session *ClientSession, content string) error {
	r := session.room
	r.mu.Lock()
	seq := r.updateSeq + 1
	r.content = content
	r.updateSeq = seq
	receivers := r.clientsSnapshotLocked()
	r.mu.Unlock()

	now := s.now().UTC()
	createdBy := session.User.ID
	updateID, err := s.newID()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(ClientUpdate{Type: "update", Content: content})
	if err != nil {
		return err
	}
	if err := s.repo.CreateUpdate(ctx, Update{
		ID:         updateID,
		DocumentID: session.DocumentID,
		UpdateSeq:  seq,
		UpdateData: payload,
		CreatedBy:  &createdBy,
		CreatedAt:  now,
	}); err != nil {
		return err
	}
	if seq%int64(s.snapshotInterval) == 0 {
		if err := s.createSnapshot(ctx, session.DocumentID, seq, content, &createdBy, now); err != nil {
			return err
		}
	}
	message := ServerMessage{
		Type:        "update",
		Content:     content,
		UpdateSeq:   seq,
		UserID:      session.User.ID,
		DisplayName: session.User.DisplayName,
	}
	for _, receiver := range receivers {
		if receiver != session {
			receiver.enqueue(message)
		}
	}
	return nil
}

func (s *Service) checkAccess(ctx context.Context, userID string, documentID string) (bool, error) {
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return false, err
	}
	if !isTextDocument(doc.FileExt) {
		return false, ErrUnsupportedFile
	}
	canView, err := s.permissions.CanView(ctx, userID, documentID)
	if err != nil {
		return false, err
	}
	if !canView {
		return false, ErrForbidden
	}
	canEdit, err := s.permissions.CanEdit(ctx, userID, documentID)
	if err != nil {
		return false, err
	}
	return canEdit, nil
}

func (s *Service) getRoom(ctx context.Context, documentID string) (*room, error) {
	s.mu.Lock()
	existing := s.rooms[documentID]
	s.mu.Unlock()
	if existing != nil {
		return existing, nil
	}
	content, seq, err := s.currentContent(ctx, documentID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing = s.rooms[documentID]; existing != nil {
		return existing, nil
	}
	r := &room{
		documentID: documentID,
		content:    content,
		updateSeq:  seq,
		clients:    map[*ClientSession]struct{}{},
	}
	s.rooms[documentID] = r
	return r, nil
}

func (s *Service) currentContent(ctx context.Context, documentID string) (string, int64, error) {
	snapshot, err := s.repo.LatestSnapshot(ctx, documentID)
	content := ""
	seq := int64(0)
	if err == nil {
		content = snapshot.Content
		seq = snapshot.VersionNo
	} else if errors.Is(err, ErrNotFound) {
		initial, initialErr := s.initialDocumentContent(ctx, documentID)
		if initialErr != nil {
			return "", 0, initialErr
		}
		content = initial
	} else {
		return "", 0, err
	}
	updates, err := s.repo.ListUpdatesAfter(ctx, documentID, seq)
	if err != nil {
		return "", 0, err
	}
	for _, update := range updates {
		var payload ClientUpdate
		if err := json.Unmarshal(update.UpdateData, &payload); err != nil {
			return "", 0, fmt.Errorf("decode markdown update: %w", err)
		}
		if payload.Type == "update" {
			content = payload.Content
			seq = update.UpdateSeq
		}
	}
	return content, seq, nil
}

func (s *Service) initialDocumentContent(ctx context.Context, documentID string) (string, error) {
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return "", err
	}
	reader, err := s.storage.GetObject(ctx, doc.StorageKey)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) createSnapshot(ctx context.Context, documentID string, versionNo int64, content string, createdBy *string, createdAt time.Time) error {
	id, err := s.newID()
	if err != nil {
		return err
	}
	return s.repo.CreateSnapshot(ctx, Snapshot{
		ID:         id,
		DocumentID: documentID,
		VersionNo:  versionNo,
		Content:    content,
		CreatedBy:  createdBy,
		CreatedAt:  createdAt,
	})
}

func (s *Service) broadcastPresence(r *room) {
	if r == nil {
		return
	}
	r.mu.Lock()
	users := r.presenceLocked()
	receivers := r.clientsSnapshotLocked()
	r.mu.Unlock()
	message := ServerMessage{Type: "presence", Users: users}
	for _, receiver := range receivers {
		receiver.enqueue(message)
	}
}

func (s *Service) presence(documentID string) []PresenceUser {
	s.mu.Lock()
	r := s.rooms[documentID]
	s.mu.Unlock()
	if r == nil {
		return []PresenceUser{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.presenceLocked()
}

func (r *room) presenceLocked() []PresenceUser {
	users := make([]PresenceUser, 0, len(r.clients))
	for client := range r.clients {
		users = append(users, PresenceUser{
			UserID:      client.User.ID,
			DisplayName: client.User.DisplayName,
			CanEdit:     client.CanEdit,
		})
	}
	return users
}

func (r *room) clientsSnapshotLocked() []*ClientSession {
	clients := make([]*ClientSession, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}

func (s *ClientSession) enqueue(message ServerMessage) {
	select {
	case s.Receive <- message:
	default:
	}
}

func isTextDocument(fileExt string) bool {
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	return ext == "md" || ext == "markdown" || ext == "txt"
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
