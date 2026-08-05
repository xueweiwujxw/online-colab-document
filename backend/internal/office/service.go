package office

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	JWTSecret       string
	DocsEditorURL   string
	SheetsEditorURL string
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
	EditorURL   string `json:"editorUrl"`
}

type wopiClaims struct {
	Subject     string   `json:"sub"`
	DisplayName string   `json:"display_name"`
	FileID      string   `json:"file_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	Kind        string   `json:"kind"`
	ExpiresAt   int64    `json:"exp"`
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
	if !isCasualSupported(doc.FileExt) || s.cfg.JWTSecret == "" {
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
	role := "viewer"
	if canEdit {
		role = "editor"
	}
	token, err := s.mintWOPIToken(currentUser, doc.ID, role, editorKind(doc.FileExt))
	if err != nil {
		return Session{}, err
	}
	return Session{
		Provider:    s.cfg.Provider,
		DocumentID:  doc.ID,
		FileExt:     strings.ToLower(doc.FileExt),
		Title:       doc.OriginalFilename,
		Mode:        mode,
		DownloadURL: base + "/api/documents/" + doc.ID + "/download",
		SaveURL:     base + "/api/documents/" + doc.ID + "/office/content",
		EditorURL:   editorURL(s.cfg, doc, token),
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

// WOPIInfo validates the short-lived token and then checks the live document
// permission. The second check is intentional: removing access revokes an
// already-issued editor session immediately instead of waiting for token expiry.
func (s *Service) WOPIInfo(ctx context.Context, token, documentID string) (document.Document, bool, string, error) {
	claims, err := s.verifyWOPIToken(token, documentID)
	if err != nil {
		return document.Document{}, false, "", err
	}
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return document.Document{}, false, "", err
	}
	canView, err := s.permissions.CanView(ctx, claims.Subject, documentID)
	if err != nil {
		return document.Document{}, false, "", err
	}
	if !canView {
		return document.Document{}, false, "", ErrForbidden
	}
	canEdit, err := s.permissions.CanEdit(ctx, claims.Subject, documentID)
	if err != nil {
		return document.Document{}, false, "", err
	}
	version := doc.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if doc.CurrentVersionID != nil {
		version = *doc.CurrentVersionID
	}
	return doc, canEdit, version, nil
}

func (s *Service) WOPIContent(ctx context.Context, token, documentID string) (document.Document, io.ReadCloser, string, error) {
	doc, _, version, err := s.WOPIInfo(ctx, token, documentID)
	if err != nil {
		return document.Document{}, nil, "", err
	}
	reader, err := s.storage.GetObject(ctx, doc.StorageKey)
	if err != nil {
		return document.Document{}, nil, "", err
	}
	return doc, reader, version, nil
}

func (s *Service) WOPISave(ctx context.Context, token, documentID string, body io.Reader) (document.Document, int64, error) {
	claims, err := s.verifyWOPIToken(token, documentID)
	if err != nil {
		return document.Document{}, 0, err
	}
	canEdit, err := s.permissions.CanEdit(ctx, claims.Subject, documentID)
	if err != nil {
		return document.Document{}, 0, err
	}
	if !canEdit {
		return document.Document{}, 0, ErrForbidden
	}
	return s.Save(ctx, user.User{ID: claims.Subject, DisplayName: claims.DisplayName}, documentID, body)
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
	case "md":
		return true
	default:
		return false
	}
}

func editorKind(ext string) string {
	if strings.EqualFold(ext, "xlsx") {
		return "sheets"
	}
	return "docs"
}

func editorURL(cfg Config, doc document.Document, token string) string {
	if editorKind(doc.FileExt) == "sheets" {
		return strings.TrimRight(cfg.SheetsEditorURL, "/") + "/?access_token=" + token
	}
	id := base64.RawURLEncoding.EncodeToString([]byte(doc.ID))
	return strings.TrimRight(cfg.DocsEditorURL, "/") + "/doc/" + id + "?access_token=" + token
}

func (s *Service) mintWOPIToken(currentUser user.User, fileID, role, kind string) (string, error) {
	if len(s.cfg.JWTSecret) < 16 {
		return "", ErrUnsupportedFile
	}
	permissions := []string{"read"}
	if role == "editor" {
		permissions = append(permissions, "write")
	}
	claims := wopiClaims{Subject: currentUser.ID, DisplayName: currentUser.DisplayName, FileID: fileID, Role: role, Permissions: permissions, Kind: kind, ExpiresAt: s.now().Add(15 * time.Minute).Unix()}
	return signJWT(s.cfg.JWTSecret, claims)
}

func (s *Service) verifyWOPIToken(token, fileID string) (wopiClaims, error) {
	claims, err := verifyJWT(s.cfg.JWTSecret, token)
	if err != nil || claims.FileID != fileID || claims.ExpiresAt <= s.now().Unix() {
		return wopiClaims{}, ErrForbidden
	}
	return claims, nil
}

func signJWT(secret string, claims wopiClaims) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signing := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signing))
	return signing + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyJWT(secret, token string) (wopiClaims, error) {
	var claims wopiClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(secret) < 16 {
		return claims, ErrForbidden
	}
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, ErrForbidden
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return claims, ErrForbidden
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrForbidden
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Subject == "" || claims.FileID == "" {
		return wopiClaims{}, ErrForbidden
	}
	return claims, nil
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
