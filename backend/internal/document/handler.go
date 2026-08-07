package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/middleware"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
	audit   AuditRecorder
}

type AuditRecorder interface {
	Record(ctx context.Context, input audit.RecordInput) error
}

type markdownResponse struct {
	Document PublicDocument `json:"document"`
	Content  string         `json:"content"`
	CanEdit  bool           `json:"canEdit"`
}

type updateMarkdownRequest struct {
	Content string `json:"content"`
}

func NewHandler(service *Service, logger *slog.Logger) Handler {
	return Handler{service: service, logger: logger}
}

func (h Handler) WithAudit(recorder AuditRecorder) Handler {
	h.audit = recorder
	return h
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	docs, err := h.service.List(r.Context(), currentUser.ID)
	if err != nil {
		h.writeError(w, "list documents failed", err)
		return
	}
	items := make([]PublicDocument, 0, len(docs))
	for _, doc := range docs {
		canManage, err := h.service.CanManage(r.Context(), currentUser.ID, doc.ID)
		if err != nil {
			h.writeError(w, "check document manage permission failed", err)
			return
		}
		canEdit, err := h.service.CanEdit(r.Context(), currentUser.ID, doc.ID)
		if err != nil {
			h.writeError(w, "check document edit permission failed", err)
			return
		}
		items = append(items, ToPublic(doc, canManage, canEdit))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	docs, err := h.service.ListAdmin(r.Context(), limit)
	if err != nil {
		h.writeError(w, "list admin documents failed", err)
		return
	}
	items := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		items = append(items, map[string]any{"id": doc.ID, "ownerId": doc.OwnerID, "title": doc.Title, "originalFilename": doc.OriginalFilename, "fileExt": doc.FileExt, "sizeBytes": doc.SizeBytes, "updatedAt": doc.UpdatedAt})
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.service.DeleteAdmin(r.Context(), r.PathValue("id")); err != nil {
		h.writeError(w, "admin delete document failed", err)
		return
	}
	actorID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{ActorUserID: &actorID, Action: "admin.document_delete", TargetType: "document", TargetID: r.PathValue("id")})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.service.MaxUploadBytes()+1024)
	if err := r.ParseMultipartForm(h.service.MaxUploadBytes()); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	doc, err := h.service.Upload(r.Context(), UploadInput{
		OwnerID:          currentUser.ID,
		OriginalFilename: header.Filename,
		HeaderMimeType:   header.Header.Get("Content-Type"),
		SizeBytes:        header.Size,
		Reader:           file,
	})
	if err != nil {
		h.writeError(w, "upload document failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionDocumentUpload,
		TargetType:  "document",
		TargetID:    doc.ID,
		Metadata: map[string]any{
			"fileExt":   doc.FileExt,
			"mimeType":  doc.MimeType,
			"sizeBytes": doc.SizeBytes,
		},
	})
	api.WriteJSON(w, http.StatusCreated, ToPublic(doc, true, true))
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	doc, err := h.service.Get(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "get document failed", err)
		return
	}
	canManage, err := h.service.CanManage(r.Context(), currentUser.ID, doc.ID)
	if err != nil {
		h.writeError(w, "check document manage permission failed", err)
		return
	}
	canEdit, err := h.service.CanEdit(r.Context(), currentUser.ID, doc.ID)
	if err != nil {
		h.writeError(w, "check document edit permission failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, ToPublic(doc, canManage, canEdit))
}

func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	download, err := h.service.Download(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "download document failed", err)
		return
	}
	defer download.Reader.Close()
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionDocumentDownload,
		TargetType:  "document",
		TargetID:    download.Document.ID,
		Metadata: map[string]any{
			"sizeBytes": download.Document.SizeBytes,
		},
	})

	w.Header().Set("Content-Type", download.Document.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.Document.SizeBytes))
	w.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": download.Document.OriginalFilename}),
	)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, download.Reader); err != nil {
		h.logger.Error("stream document failed", "error", err)
	}
}

func (h Handler) DownloadVersion(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	download, err := h.service.DownloadVersion(r.Context(), currentUser.ID, r.PathValue("id"), r.PathValue("versionId"))
	if err != nil {
		h.writeError(w, "download document version failed", err)
		return
	}
	defer download.Reader.Close()

	w.Header().Set("Content-Type", download.Document.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.Version.SizeBytes))
	w.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": download.Document.OriginalFilename}),
	)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, download.Reader); err != nil {
		h.logger.Error("stream document version failed", "error", err)
	}
}

func (h Handler) GetMarkdown(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	markdown, err := h.service.GetMarkdown(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "get markdown document failed", err)
		return
	}
	canManage, err := h.service.CanManage(r.Context(), currentUser.ID, markdown.Document.ID)
	if err != nil {
		h.writeError(w, "check document manage permission failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, markdownResponse{
		Document: ToPublic(markdown.Document, canManage, markdown.CanEdit),
		Content:  markdown.Content,
		CanEdit:  markdown.CanEdit,
	})
}

func (h Handler) UpdateMarkdown(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var input updateMarkdownRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	markdown, err := h.service.SaveMarkdown(r.Context(), currentUser.ID, r.PathValue("id"), input.Content)
	if err != nil {
		h.writeError(w, "save markdown document failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionMarkdownSave,
		TargetType:  "document",
		TargetID:    markdown.Document.ID,
		Metadata: map[string]any{
			"sizeBytes": markdown.Document.SizeBytes,
		},
	})
	canManage, err := h.service.CanManage(r.Context(), currentUser.ID, markdown.Document.ID)
	if err != nil {
		h.writeError(w, "check document manage permission failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, markdownResponse{
		Document: ToPublic(markdown.Document, canManage, markdown.CanEdit),
		Content:  markdown.Content,
		CanEdit:  markdown.CanEdit,
	})
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.service.Delete(r.Context(), currentUser.ID, r.PathValue("id")); err != nil {
		h.writeError(w, "delete document failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionDocumentDelete,
		TargetType:  "document",
		TargetID:    r.PathValue("id"),
	})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Versions(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	versions, err := h.service.Versions(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "list document versions failed", err)
		return
	}
	items := make([]PublicVersion, 0, len(versions))
	for _, version := range versions {
		items = append(items, VersionToPublic(version))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	version, err := h.service.RestoreVersion(r.Context(), currentUser.ID, r.PathValue("id"), r.PathValue("versionId"))
	if err != nil {
		h.writeError(w, "restore document version failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionVersionRestore,
		TargetType:  "document",
		TargetID:    version.DocumentID,
		Metadata: map[string]any{
			"sourceVersionId": r.PathValue("versionId"),
			"newVersionId":    version.ID,
		},
	})
	api.WriteJSON(w, http.StatusCreated, VersionToPublic(version))
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrInvalidFile):
		api.WriteError(w, http.StatusBadRequest, "invalid file")
	case errors.Is(err, ErrFileTooLarge):
		api.WriteError(w, http.StatusRequestEntityTooLarge, "file too large")
	case errors.Is(err, ErrUnsupportedType):
		api.WriteError(w, http.StatusBadRequest, "unsupported file type")
	default:
		h.logger.Error(logMessage, "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h Handler) recordAudit(r *http.Request, input audit.RecordInput) {
	if h.audit == nil {
		return
	}
	input.IPAddr, input.UserAgent = audit.RequestInfo(r)
	if err := h.audit.Record(r.Context(), input); err != nil {
		h.logger.Error("audit log failed", "error", err)
	}
}
