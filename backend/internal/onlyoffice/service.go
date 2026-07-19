package onlyoffice

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/storage"
	"online-colab-document/backend/internal/user"
)

var (
	ErrDisabled         = errors.New("onlyoffice disabled")
	ErrForbidden        = errors.New("forbidden")
	ErrUnsupportedFile  = errors.New("unsupported office document")
	ErrInvalidCallback  = errors.New("invalid onlyoffice callback")
	ErrCallbackTooLarge = errors.New("onlyoffice callback file too large")
)

type Config struct {
	Enabled          bool
	PublicURL        string
	JWTSecret        string
	PublicAPIURL     string
	CallbackBaseURL  string
	MaxDownloadBytes int64
	DownloadTimeout  time.Duration
	PresignedURLTTL  time.Duration
}

type DocumentRepository interface {
	FindByID(ctx context.Context, id string) (document.Document, error)
	AddVersion(ctx context.Context, documentID string, version document.Version, documentKey string, updatedAt time.Time) (bool, error)
	HasOnlyOfficeSave(ctx context.Context, documentID string, documentKey string) (bool, error)
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
	httpClient  *http.Client
	now         func() time.Time
	newID       func() (string, error)
}

type EditorConfig struct {
	DocumentServerURL string         `json:"documentServerUrl"`
	Document          EditorDocument `json:"document"`
	DocumentType      string         `json:"documentType"`
	EditorConfig      EditorSettings `json:"editorConfig"`
	Type              string         `json:"type"`
	Token             string         `json:"token,omitempty"`
}

type EditorDocument struct {
	FileType string `json:"fileType"`
	Key      string `json:"key"`
	Title    string `json:"title"`
	URL      string `json:"url"`
}

type EditorSettings struct {
	Mode        string     `json:"mode"`
	CallbackURL string     `json:"callbackUrl"`
	User        EditorUser `json:"user"`
}

type EditorUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CallbackRequest struct {
	Key    string `json:"key"`
	Status int    `json:"status"`
	URL    string `json:"url"`
	Token  string `json:"token"`
}

type CallbackResponse struct {
	Error int `json:"error"`
}

type Download struct {
	Document document.Document
	Reader   io.ReadCloser
}

type downloadTicket struct {
	DocumentID string `json:"documentId"`
	Key        string `json:"key"`
	StorageKey string `json:"storageKey"`
	ExpiresAt  int64  `json:"exp"`
}

func NewService(cfg Config, documents DocumentRepository, permissions PermissionService, objectStorage storage.Storage) *Service {
	if cfg.MaxDownloadBytes <= 0 {
		cfg.MaxDownloadBytes = 50 << 20
	}
	if cfg.DownloadTimeout <= 0 {
		cfg.DownloadTimeout = 10 * time.Second
	}
	if cfg.PresignedURLTTL <= 0 {
		cfg.PresignedURLTTL = 10 * time.Minute
	}
	return &Service{
		cfg:         cfg,
		documents:   documents,
		permissions: permissions,
		storage:     objectStorage,
		httpClient:  &http.Client{Timeout: cfg.DownloadTimeout},
		now:         time.Now,
		newID:       newUUID,
	}
}

func (s *Service) Config(ctx context.Context, currentUser user.User, documentID string) (EditorConfig, error) {
	if !s.cfg.Enabled {
		return EditorConfig{}, ErrDisabled
	}
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return EditorConfig{}, err
	}
	if !isOfficeDocument(doc.FileExt) {
		return EditorConfig{}, ErrUnsupportedFile
	}
	canView, err := s.permissions.CanView(ctx, currentUser.ID, documentID)
	if err != nil {
		return EditorConfig{}, err
	}
	if !canView {
		return EditorConfig{}, ErrForbidden
	}
	canEdit, err := s.permissions.CanEdit(ctx, currentUser.ID, documentID)
	if err != nil {
		return EditorConfig{}, err
	}
	mode := "view"
	if canEdit {
		mode = "edit"
	}
	downloadURL, err := s.downloadURL(doc)
	if err != nil {
		return EditorConfig{}, err
	}
	cfg := EditorConfig{
		DocumentServerURL: s.cfg.PublicURL,
		DocumentType:      documentType(doc.FileExt),
		Type:              "desktop",
		Document: EditorDocument{
			FileType: doc.FileExt,
			Key:      documentKey(doc),
			Title:    doc.OriginalFilename,
			URL:      downloadURL,
		},
		EditorConfig: EditorSettings{
			Mode:        mode,
			CallbackURL: s.callbackURL(doc.ID),
			User: EditorUser{
				ID:   currentUser.ID,
				Name: currentUser.DisplayName,
			},
		},
	}
	if s.cfg.JWTSecret != "" {
		token, err := signJWT(cfg, s.cfg.JWTSecret)
		if err != nil {
			return EditorConfig{}, err
		}
		cfg.Token = token
	}
	return cfg, nil
}

func (s *Service) Download(ctx context.Context, documentID string, token string) (Download, error) {
	if !s.cfg.Enabled {
		return Download{}, ErrDisabled
	}
	if token == "" || s.cfg.JWTSecret == "" {
		return Download{}, ErrInvalidToken
	}
	var ticket downloadTicket
	if err := verifyJWT(token, s.cfg.JWTSecret, &ticket); err != nil {
		return Download{}, err
	}
	if ticket.ExpiresAt <= s.now().Unix() || ticket.DocumentID != documentID {
		return Download{}, ErrInvalidToken
	}
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return Download{}, err
	}
	if !isOfficeDocument(doc.FileExt) {
		return Download{}, ErrUnsupportedFile
	}
	if ticket.Key != documentKey(doc) || ticket.StorageKey != doc.StorageKey {
		return Download{}, ErrInvalidToken
	}
	reader, err := s.storage.GetObject(ctx, doc.StorageKey)
	if err != nil {
		return Download{}, err
	}
	return Download{Document: doc, Reader: reader}, nil
}

func (s *Service) Callback(ctx context.Context, documentID string, callback CallbackRequest, bearerToken string) (CallbackResponse, error) {
	if !s.cfg.Enabled {
		return CallbackResponse{}, ErrDisabled
	}
	if s.cfg.JWTSecret != "" {
		token := callback.Token
		if token == "" {
			token = strings.TrimPrefix(bearerToken, "Bearer ")
		}
		var claims CallbackRequest
		if err := verifyJWT(token, s.cfg.JWTSecret, &claims); err != nil {
			return CallbackResponse{}, err
		}
		callback.Key = claims.Key
		callback.Status = claims.Status
		callback.URL = claims.URL
	}
	doc, err := s.documents.FindByID(ctx, documentID)
	if err != nil {
		return CallbackResponse{}, err
	}
	currentKey := documentKey(doc)
	if callback.Key != currentKey {
		alreadySaved, err := s.documents.HasOnlyOfficeSave(ctx, documentID, callback.Key)
		if err != nil {
			return CallbackResponse{}, err
		}
		if alreadySaved {
			return CallbackResponse{Error: 0}, nil
		}
		return CallbackResponse{}, ErrInvalidCallback
	}
	if callback.Status != 2 && callback.Status != 6 {
		return CallbackResponse{Error: 0}, nil
	}
	if callback.URL == "" {
		return CallbackResponse{}, ErrInvalidCallback
	}
	versionID, err := s.newID()
	if err != nil {
		return CallbackResponse{}, err
	}
	storageKey := fmt.Sprintf("documents/%s/versions/%s/%s", doc.ID, versionID, doc.OriginalFilename)
	size, err := s.downloadToStorage(ctx, callback.URL, storageKey, doc.MimeType)
	if err != nil {
		return CallbackResponse{}, err
	}
	now := s.now().UTC()
	version := document.Version{
		ID:         versionID,
		DocumentID: doc.ID,
		StorageKey: storageKey,
		SizeBytes:  size,
		CreatedAt:  now,
	}
	inserted, err := s.documents.AddVersion(ctx, doc.ID, version, callback.Key, now)
	if err != nil {
		_ = s.storage.DeleteObject(ctx, storageKey)
		return CallbackResponse{}, err
	}
	if !inserted {
		_ = s.storage.DeleteObject(ctx, storageKey)
	}
	return CallbackResponse{Error: 0}, nil
}

func (s *Service) downloadToStorage(ctx context.Context, sourceURL string, storageKey string, contentType string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, ErrInvalidCallback
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, ErrInvalidCallback
	}
	limited := &limitedReader{reader: resp.Body, remaining: s.cfg.MaxDownloadBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return 0, err
	}
	if int64(len(data)) > s.cfg.MaxDownloadBytes {
		return 0, ErrCallbackTooLarge
	}
	if err := s.storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
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

func (s *Service) callbackURL(documentID string) string {
	base := strings.TrimRight(s.cfg.CallbackBaseURL, "/")
	if base == "" {
		base = strings.TrimRight(s.cfg.PublicAPIURL, "/")
	}
	return base + "/api/onlyoffice/callback/" + documentID
}

func (s *Service) downloadURL(doc document.Document) (string, error) {
	if s.cfg.JWTSecret == "" {
		return "", ErrInvalidToken
	}
	base := strings.TrimRight(s.cfg.CallbackBaseURL, "/")
	if base == "" {
		base = strings.TrimRight(s.cfg.PublicAPIURL, "/")
	}
	if base == "" {
		return "", ErrInvalidToken
	}
	ticket := downloadTicket{
		DocumentID: doc.ID,
		Key:        documentKey(doc),
		StorageKey: doc.StorageKey,
		ExpiresAt:  s.now().Add(s.cfg.PresignedURLTTL).Unix(),
	}
	token, err := signJWT(ticket, s.cfg.JWTSecret)
	if err != nil {
		return "", err
	}
	downloadURL, err := url.Parse(base + "/api/onlyoffice/download/" + doc.ID)
	if err != nil {
		return "", err
	}
	query := downloadURL.Query()
	query.Set("token", token)
	downloadURL.RawQuery = query.Encode()
	return downloadURL.String(), nil
}

func documentKey(doc document.Document) string {
	version := "initial"
	if doc.CurrentVersionID != nil && *doc.CurrentVersionID != "" {
		version = *doc.CurrentVersionID
	}
	return "doc-" + doc.ID + "-" + version
}

func isOfficeDocument(ext string) bool {
	switch strings.ToLower(ext) {
	case "doc", "docx", "xls", "xlsx":
		return true
	default:
		return false
	}
}

func documentType(ext string) string {
	switch strings.ToLower(ext) {
	case "xls", "xlsx":
		return "cell"
	default:
		return "word"
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
