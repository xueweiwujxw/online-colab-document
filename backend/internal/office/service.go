package office

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/storage"
	"online-colab-document/backend/internal/user"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrUnsupportedFile = errors.New("unsupported office document")
	ErrTooLarge        = errors.New("office document too large")
)

type Config struct {
	Provider        string
	PublicAPIURL    string
	CollabPublicURL string
	CollabEnabled   bool
	MaxUploadBytes  int64
}

type DocumentRepository interface {
	FindByID(ctx context.Context, id string) (document.Document, error)
	AddDocumentVersion(ctx context.Context, documentID string, version document.Version, updatedAt time.Time) error
}

type PermissionService interface {
	CanView(ctx context.Context, userID string, documentID string) (bool, error)
	CanEdit(ctx context.Context, userID string, documentID string) (bool, error)
}

type Service struct {
	cfg         Config
	documents   DocumentRepository
	permissions PermissionService
	storage     storage.Storage
	now         func() time.Time
	newID       func() (string, error)
}

type Session struct {
	Provider    string `json:"provider"`
	DocumentID  string `json:"documentId"`
	FileExt     string `json:"fileExt"`
	Title       string `json:"title"`
	Mode        string `json:"mode"`
	DownloadURL string `json:"downloadUrl"`
	SaveURL     string `json:"saveUrl"`
}

type CollabSession struct {
	Enabled    bool   `json:"enabled"`
	DocumentID string `json:"documentId"`
	FileExt    string `json:"fileExt"`
	Room       string `json:"room"`
	Role       string `json:"role"`
	ServerURL  string `json:"serverUrl,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func NewService(cfg Config, documents DocumentRepository, permissions PermissionService, objectStorage storage.Storage) *Service {
	if cfg.Provider == "" {
		cfg.Provider = "casual"
	}
	if cfg.MaxUploadBytes <= 0 {
		cfg.MaxUploadBytes = 50 << 20
	}
	return &Service{
		cfg:         cfg,
		documents:   documents,
		permissions: permissions,
		storage:     objectStorage,
		now:         time.Now,
		newID:       newUUID,
	}
}

func (s *Service) Session(ctx context.Context, currentUser user.User, documentID string) (Session, error) {
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return Session{}, err
	}
	if !isCasualSupported(doc.FileExt) {
		return Session{}, ErrUnsupportedFile
	}
	canView, err := s.permissions.CanView(ctx, currentUser.ID, documentID)
	if err != nil {
		return Session{}, err
	}
	if !canView {
		return Session{}, ErrForbidden
	}
	canEdit, err := s.permissions.CanEdit(ctx, currentUser.ID, documentID)
	if err != nil {
		return Session{}, err
	}
	mode := "view"
	if canEdit {
		mode = "edit"
	}
	base := strings.TrimRight(s.cfg.PublicAPIURL, "/")
	return Session{
		Provider:    s.cfg.Provider,
		DocumentID:  doc.ID,
		FileExt:     strings.ToLower(doc.FileExt),
		Title:       doc.OriginalFilename,
		Mode:        mode,
		DownloadURL: base + "/api/documents/" + doc.ID + "/download",
		SaveURL:     base + "/api/documents/" + doc.ID + "/office/content",
	}, nil
}

func (s *Service) CollabSession(ctx context.Context, currentUser user.User, documentID string) (CollabSession, error) {
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return CollabSession{}, err
	}
	if strings.ToLower(doc.FileExt) != "xlsx" {
		return CollabSession{}, ErrUnsupportedFile
	}
	canView, err := s.permissions.CanView(ctx, currentUser.ID, documentID)
	if err != nil {
		return CollabSession{}, err
	}
	if !canView {
		return CollabSession{}, ErrForbidden
	}
	canEdit, err := s.permissions.CanEdit(ctx, currentUser.ID, documentID)
	if err != nil {
		return CollabSession{}, err
	}
	role := "view"
	if canEdit {
		role = "write"
	}
	serverURL := strings.TrimRight(s.cfg.CollabPublicURL, "/")
	session := CollabSession{
		Enabled:    s.cfg.CollabEnabled && serverURL != "",
		DocumentID: doc.ID,
		FileExt:    strings.ToLower(doc.FileExt),
		Room:       "office:" + doc.ID,
		Role:       role,
		ServerURL:  serverURL,
	}
	if !session.Enabled {
		session.Reason = "office collab service is not configured"
	}
	return session, nil
}

func (s *Service) Save(ctx context.Context, currentUser user.User, documentID string, body io.Reader) (document.Document, int64, error) {
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return document.Document{}, 0, err
	}
	if !isCasualSupported(doc.FileExt) {
		return document.Document{}, 0, ErrUnsupportedFile
	}
	canEdit, err := s.permissions.CanEdit(ctx, currentUser.ID, documentID)
	if err != nil {
		return document.Document{}, 0, err
	}
	if !canEdit {
		return document.Document{}, 0, ErrForbidden
	}
	limited := &limitedReader{reader: body, remaining: s.cfg.MaxUploadBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return document.Document{}, 0, err
	}
	if int64(len(data)) > s.cfg.MaxUploadBytes {
		return document.Document{}, 0, ErrTooLarge
	}
	versionID, err := s.newID()
	if err != nil {
		return document.Document{}, 0, err
	}
	storageKey := fmt.Sprintf("documents/%s/versions/%s/%s", doc.ID, versionID, doc.OriginalFilename)
	if err := s.storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), doc.MimeType); err != nil {
		return document.Document{}, 0, err
	}
	now := s.now().UTC()
	createdBy := currentUser.ID
	version := document.Version{
		ID:         versionID,
		DocumentID: doc.ID,
		StorageKey: storageKey,
		SizeBytes:  int64(len(data)),
		CreatedBy:  &createdBy,
		CreatedAt:  now,
	}
	if err := s.documents.AddDocumentVersion(ctx, doc.ID, version, now); err != nil {
		_ = s.storage.DeleteObject(ctx, storageKey)
		return document.Document{}, 0, err
	}
	return doc, int64(len(data)), nil
}

type limitedReader struct {
	reader    io.Reader
	remaining int64
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:int(r.remaining)]
	}
	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	return n, err
}

func isCasualSupported(ext string) bool {
	switch strings.ToLower(ext) {
	case "docx", "xlsx":
		return true
	default:
		return false
	}
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
