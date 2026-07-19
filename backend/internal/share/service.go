package share

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/storage"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidInput    = errors.New("invalid share input")
	ErrExpired         = errors.New("share link expired")
	ErrDisabled        = errors.New("share link disabled")
	ErrUnsupportedFile = errors.New("unsupported shared file")
)

type DocumentRepository interface {
	FindByID(ctx context.Context, id string) (document.Document, error)
	AddDocumentVersion(ctx context.Context, documentID string, version document.Version, updatedAt time.Time) error
}

type PermissionService interface {
	CanShare(ctx context.Context, userID string, documentID string) (bool, error)
}

type CreateInput struct {
	ActorID    string
	DocumentID string
	Permission string
	ExpiresAt  *time.Time
}

type Download struct {
	Document document.Document
	Reader   io.ReadCloser
}

type Service struct {
	repo         Repository
	documents    DocumentRepository
	permissions  PermissionService
	storage      storage.Storage
	publicAppURL string
	now          func() time.Time
	newID        func() (string, error)
	newToken     func() (string, error)
}

func NewService(repo Repository, documents DocumentRepository, permissions PermissionService, objectStorage storage.Storage, publicAppURL string) *Service {
	return &Service{
		repo:         repo,
		documents:    documents,
		permissions:  permissions,
		storage:      objectStorage,
		publicAppURL: strings.TrimRight(publicAppURL, "/"),
		now:          time.Now,
		newID:        newUUID,
		newToken:     newToken,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResponse, error) {
	if input.ActorID == "" || input.DocumentID == "" {
		return CreateResponse{}, ErrInvalidInput
	}
	if input.Permission != PermissionViewer && input.Permission != PermissionEditor {
		return CreateResponse{}, ErrInvalidInput
	}
	canShare, err := s.permissions.CanShare(ctx, input.ActorID, input.DocumentID)
	if err != nil {
		return CreateResponse{}, err
	}
	if !canShare {
		return CreateResponse{}, ErrForbidden
	}
	if _, err := s.documents.FindByID(ctx, input.DocumentID); err != nil {
		return CreateResponse{}, err
	}
	id, err := s.newID()
	if err != nil {
		return CreateResponse{}, err
	}
	token, err := s.newToken()
	if err != nil {
		return CreateResponse{}, err
	}
	createdBy := input.ActorID
	link := Link{
		ID:         id,
		DocumentID: input.DocumentID,
		TokenHash:  HashToken(token),
		Permission: input.Permission,
		ExpiresAt:  input.ExpiresAt,
		Disabled:   false,
		CreatedBy:  &createdBy,
		CreatedAt:  s.now().UTC(),
	}
	if err := s.repo.Create(ctx, link); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{
		PublicLink: ToPublic(link),
		Token:      token,
		URL:        s.shareURL(token),
	}, nil
}

func (s *Service) List(ctx context.Context, actorID string, documentID string) ([]Link, error) {
	if actorID == "" || documentID == "" {
		return nil, ErrInvalidInput
	}
	canShare, err := s.permissions.CanShare(ctx, actorID, documentID)
	if err != nil {
		return nil, err
	}
	if !canShare {
		return nil, ErrForbidden
	}
	return s.repo.ListForDocument(ctx, documentID)
}

func (s *Service) Disable(ctx context.Context, actorID string, linkID string) error {
	if actorID == "" || linkID == "" {
		return ErrInvalidInput
	}
	link, err := s.repo.FindByID(ctx, linkID)
	if err != nil {
		return err
	}
	canShare, err := s.permissions.CanShare(ctx, actorID, link.DocumentID)
	if err != nil {
		return err
	}
	if !canShare {
		return ErrForbidden
	}
	return s.repo.Disable(ctx, linkID)
}

func (s *Service) Access(ctx context.Context, token string) (AccessResponse, error) {
	link, doc, err := s.validLink(ctx, token)
	if err != nil {
		return AccessResponse{}, err
	}
	response := AccessResponse{
		Document: document.ToPublic(doc, false),
		CanEdit:  link.Permission == PermissionEditor,
	}
	if isMarkdownDocument(doc.FileExt) {
		content, err := s.readDocumentContent(ctx, doc)
		if err != nil {
			return AccessResponse{}, err
		}
		response.Content = content
	}
	return response, nil
}

func (s *Service) Download(ctx context.Context, token string) (Download, error) {
	_, doc, err := s.validLink(ctx, token)
	if err != nil {
		return Download{}, err
	}
	reader, err := s.storage.GetObject(ctx, doc.StorageKey)
	if err != nil {
		return Download{}, err
	}
	return Download{Document: doc, Reader: reader}, nil
}

func (s *Service) SaveMarkdown(ctx context.Context, token string, content string) (AccessResponse, error) {
	link, doc, err := s.validLink(ctx, token)
	if err != nil {
		return AccessResponse{}, err
	}
	if link.Permission != PermissionEditor {
		return AccessResponse{}, ErrForbidden
	}
	if !isMarkdownDocument(doc.FileExt) {
		return AccessResponse{}, ErrUnsupportedFile
	}
	versionID, err := s.newID()
	if err != nil {
		return AccessResponse{}, err
	}
	data := []byte(content)
	storageKey := storageKey(doc.ID, versionID, doc.OriginalFilename)
	if err := s.storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), doc.MimeType); err != nil {
		return AccessResponse{}, err
	}
	now := s.now().UTC()
	version := document.Version{
		ID:         versionID,
		DocumentID: doc.ID,
		StorageKey: storageKey,
		SizeBytes:  int64(len(data)),
		CreatedAt:  now,
	}
	if err := s.documents.AddDocumentVersion(ctx, doc.ID, version, now); err != nil {
		_ = s.storage.DeleteObject(ctx, storageKey)
		return AccessResponse{}, err
	}
	updated, err := s.documents.FindByID(ctx, doc.ID)
	if err != nil {
		return AccessResponse{}, err
	}
	return AccessResponse{
		Document: document.ToPublic(updated, false),
		Content:  content,
		CanEdit:  true,
	}, nil
}

func (s *Service) validLink(ctx context.Context, token string) (Link, document.Document, error) {
	if token == "" {
		return Link{}, document.Document{}, ErrNotFound
	}
	link, err := s.repo.FindByTokenHash(ctx, HashToken(token))
	if err != nil {
		return Link{}, document.Document{}, err
	}
	if link.Disabled {
		return Link{}, document.Document{}, ErrDisabled
	}
	if link.ExpiresAt != nil && !link.ExpiresAt.After(s.now().UTC()) {
		return Link{}, document.Document{}, ErrExpired
	}
	doc, err := s.documents.FindByID(ctx, link.DocumentID)
	if err != nil {
		return Link{}, document.Document{}, err
	}
	return link, doc, nil
}

func (s *Service) readDocumentContent(ctx context.Context, doc document.Document) (string, error) {
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

func (s *Service) shareURL(token string) string {
	if s.publicAppURL == "" {
		return "/share/" + token
	}
	return s.publicAppURL + "/share/" + token
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
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

func isMarkdownDocument(fileExt string) bool {
	ext := strings.ToLower(strings.TrimPrefix(fileExt, "."))
	return ext == "md" || ext == "markdown"
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func storageKey(documentID string, versionID string, filename string) string {
	safe := unsafeFilenameChars.ReplaceAllString(filepath.Base(filename), "_")
	safe = strings.Trim(safe, "._-")
	if safe == "" {
		safe = "document"
	}
	return fmt.Sprintf("documents/%s/versions/%s/%s", documentID, versionID, safe)
}
