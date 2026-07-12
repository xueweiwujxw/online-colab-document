package document

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"online-colab-document/backend/internal/storage"
)

const defaultMaxUploadBytes int64 = 50 << 20

var (
	ErrInvalidFile     = errors.New("invalid file")
	ErrFileTooLarge    = errors.New("file too large")
	ErrUnsupportedType = errors.New("unsupported file type")
	ErrForbidden       = errors.New("forbidden")
)

type Service struct {
	repo           Repository
	storage        storage.Storage
	permissions    PermissionService
	maxUploadBytes int64
	now            func() time.Time
	newID          func() (string, error)
}

type PermissionService interface {
	CanView(ctx context.Context, userID string, documentID string) (bool, error)
	CanManage(ctx context.Context, userID string, documentID string) (bool, error)
	CanDelete(ctx context.Context, userID string, documentID string) (bool, error)
}

type UploadInput struct {
	OwnerID          string
	OriginalFilename string
	HeaderMimeType   string
	SizeBytes        int64
	Reader           io.Reader
}

type Download struct {
	Document Document
	Reader   io.ReadCloser
}

func NewService(repo Repository, objectStorage storage.Storage, permissions PermissionService, maxUploadBytes int64) *Service {
	if maxUploadBytes <= 0 {
		maxUploadBytes = defaultMaxUploadBytes
	}
	return &Service{
		repo:           repo,
		storage:        objectStorage,
		permissions:    permissions,
		maxUploadBytes: maxUploadBytes,
		now:            time.Now,
		newID:          newUUID,
	}
}

func (s *Service) MaxUploadBytes() int64 {
	return s.maxUploadBytes
}

func (s *Service) Upload(ctx context.Context, input UploadInput) (Document, error) {
	if input.OwnerID == "" || input.Reader == nil || strings.TrimSpace(input.OriginalFilename) == "" {
		return Document{}, ErrInvalidFile
	}
	if input.SizeBytes <= 0 {
		return Document{}, ErrInvalidFile
	}
	if input.SizeBytes > s.maxUploadBytes {
		return Document{}, ErrFileTooLarge
	}

	metadata, err := validateFile(input.OriginalFilename, input.HeaderMimeType, input.Reader)
	if err != nil {
		return Document{}, err
	}

	documentID, err := s.newID()
	if err != nil {
		return Document{}, err
	}
	versionID, err := s.newID()
	if err != nil {
		return Document{}, err
	}
	storageKey := storageKey(documentID, versionID, input.OriginalFilename)
	if err := s.storage.PutObject(ctx, storageKey, metadata.Reader, input.SizeBytes, metadata.MimeType); err != nil {
		return Document{}, err
	}

	now := s.now().UTC()
	createdBy := input.OwnerID
	doc := Document{
		ID:               documentID,
		OwnerID:          input.OwnerID,
		Title:            titleFromFilename(input.OriginalFilename),
		OriginalFilename: filepath.Base(input.OriginalFilename),
		FileExt:          metadata.Extension,
		MimeType:         metadata.MimeType,
		StorageKey:       storageKey,
		CurrentVersionID: &versionID,
		SizeBytes:        input.SizeBytes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	version := Version{
		ID:         versionID,
		DocumentID: documentID,
		VersionNo:  1,
		StorageKey: storageKey,
		SizeBytes:  input.SizeBytes,
		CreatedBy:  &createdBy,
		CreatedAt:  now,
	}
	if err := s.repo.CreateWithVersion(ctx, doc, version); err != nil {
		_ = s.storage.DeleteObject(ctx, storageKey)
		return Document{}, err
	}
	return doc, nil
}

func (s *Service) List(ctx context.Context, ownerID string) ([]Document, error) {
	if ownerID == "" {
		return nil, ErrForbidden
	}
	if s.permissions == nil {
		return s.repo.ListByOwner(ctx, ownerID)
	}
	return s.repo.ListAccessible(ctx, ownerID)
}

func (s *Service) Get(ctx context.Context, userID string, id string) (Document, error) {
	if userID == "" || id == "" {
		return Document{}, ErrForbidden
	}
	if err := s.requireView(ctx, userID, id); err != nil {
		return Document{}, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *Service) Download(ctx context.Context, userID string, id string) (Download, error) {
	doc, err := s.Get(ctx, userID, id)
	if err != nil {
		return Download{}, err
	}
	reader, err := s.storage.GetObject(ctx, doc.StorageKey)
	if err != nil {
		return Download{}, err
	}
	return Download{Document: doc, Reader: reader}, nil
}

func (s *Service) Delete(ctx context.Context, userID string, id string) error {
	if userID == "" || id == "" {
		return ErrForbidden
	}
	if s.permissions != nil {
		canDelete, err := s.permissions.CanDelete(ctx, userID, id)
		if err != nil {
			return err
		}
		if !canDelete {
			return ErrForbidden
		}
	}
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if s.permissions == nil && doc.OwnerID != userID {
		return ErrForbidden
	}
	return s.repo.SoftDeleteForOwner(ctx, id, doc.OwnerID, s.now().UTC())
}

func (s *Service) Versions(ctx context.Context, userID string, id string) ([]Version, error) {
	if userID == "" || id == "" {
		return nil, ErrForbidden
	}
	if err := s.requireView(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.repo.ListVersions(ctx, id)
}

func (s *Service) CanManage(ctx context.Context, userID string, id string) (bool, error) {
	if s.permissions == nil {
		doc, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return false, err
		}
		return doc.OwnerID == userID, nil
	}
	return s.permissions.CanManage(ctx, userID, id)
}

func (s *Service) requireView(ctx context.Context, userID string, id string) error {
	if s.permissions == nil {
		doc, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if doc.OwnerID != userID {
			return ErrForbidden
		}
		return nil
	}
	canView, err := s.permissions.CanView(ctx, userID, id)
	if err != nil {
		return err
	}
	if !canView {
		return ErrForbidden
	}
	return nil
}

type fileMetadata struct {
	Extension string
	MimeType  string
	Reader    io.Reader
}

func validateFile(filename string, headerMimeType string, reader io.Reader) (fileMetadata, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if ext == "" {
		return fileMetadata{}, ErrUnsupportedType
	}
	allowed, ok := allowedFileTypes[ext]
	if !ok {
		return fileMetadata{}, ErrUnsupportedType
	}

	var sniff [512]byte
	n, err := io.ReadFull(reader, sniff[:])
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return fileMetadata{}, ErrInvalidFile
	}
	detected := http.DetectContentType(sniff[:n])
	normalizedHeader := normalizeMime(headerMimeType)
	if !allowed.accepts(normalizedHeader) && !allowed.accepts(detected) {
		return fileMetadata{}, ErrUnsupportedType
	}
	contentType := normalizedHeader
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = allowed.preferred
	}
	return fileMetadata{
		Extension: ext,
		MimeType:  contentType,
		Reader:    io.MultiReader(bytes.NewReader(sniff[:n]), reader),
	}, nil
}

type allowedType struct {
	preferred string
	mimes     map[string]struct{}
}

func (t allowedType) accepts(mimeType string) bool {
	mimeType = normalizeMime(mimeType)
	if mimeType == "" {
		return false
	}
	_, ok := t.mimes[mimeType]
	return ok
}

var allowedFileTypes = map[string]allowedType{
	"doc":      newAllowedType("application/msword", "application/octet-stream"),
	"docx":     newAllowedType("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip", "application/octet-stream"),
	"xls":      newAllowedType("application/vnd.ms-excel", "application/octet-stream"),
	"xlsx":     newAllowedType("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip", "application/octet-stream"),
	"md":       newAllowedType("text/markdown", "text/plain; charset=utf-8", "text/plain", "application/octet-stream"),
	"markdown": newAllowedType("text/markdown", "text/plain; charset=utf-8", "text/plain", "application/octet-stream"),
}

func newAllowedType(preferred string, alternates ...string) allowedType {
	values := append([]string{preferred}, alternates...)
	mimes := make(map[string]struct{}, len(values))
	for _, value := range values {
		mimes[normalizeMime(value)] = struct{}{}
	}
	return allowedType{preferred: preferred, mimes: mimes}
}

func normalizeMime(value string) string {
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	return strings.ToLower(mediaType)
}

func titleFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	title := strings.TrimSuffix(base, ext)
	title = strings.TrimSpace(title)
	if title == "" {
		return base
	}
	return title
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
