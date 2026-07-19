package share

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/document"
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

func NewHandler(service *Service, logger *slog.Logger) Handler {
	return Handler{service: service, logger: logger}
}

func (h Handler) WithAudit(recorder AuditRecorder) Handler {
	h.audit = recorder
	return h
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req struct {
		Permission string     `json:"permission"`
		ExpiresAt  *time.Time `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	response, err := h.service.Create(r.Context(), CreateInput{
		ActorID:    currentUser.ID,
		DocumentID: r.PathValue("id"),
		Permission: req.Permission,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		h.writeError(w, "create share link failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionShareCreate,
		TargetType:  "document",
		TargetID:    response.DocumentID,
		Metadata: map[string]any{
			"shareLinkId": response.ID,
			"permission":  response.Permission,
			"expiresAt":   response.ExpiresAt,
		},
	})
	api.WriteJSON(w, http.StatusCreated, response)
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	links, err := h.service.List(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "list share links failed", err)
		return
	}
	items := make([]PublicLink, 0, len(links))
	for _, link := range links {
		items = append(items, ToPublic(link))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) Disable(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.service.Disable(r.Context(), currentUser.ID, r.PathValue("id")); err != nil {
		h.writeError(w, "disable share link failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionShareDisable,
		TargetType:  "share_link",
		TargetID:    r.PathValue("id"),
	})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Access(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.Access(r.Context(), r.PathValue("token"))
	if err != nil {
		h.writeError(w, "access share link failed", err)
		return
	}
	h.recordAudit(r, audit.RecordInput{
		Action:     audit.ActionShareAccess,
		TargetType: "document",
		TargetID:   response.Document.ID,
		Metadata: map[string]any{
			"canEdit": response.CanEdit,
		},
	})
	api.WriteJSON(w, http.StatusOK, response)
}

func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	download, err := h.service.Download(r.Context(), r.PathValue("token"))
	if err != nil {
		h.writeError(w, "download shared document failed", err)
		return
	}
	defer download.Reader.Close()
	h.recordAudit(r, audit.RecordInput{
		Action:     audit.ActionShareDownload,
		TargetType: "document",
		TargetID:   download.Document.ID,
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
		h.logger.Error("stream shared document failed", "error", err)
	}
}

func (h Handler) SaveMarkdown(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	response, err := h.service.SaveMarkdown(r.Context(), r.PathValue("token"), req.Content)
	if err != nil {
		h.writeError(w, "save shared markdown failed", err)
		return
	}
	h.recordAudit(r, audit.RecordInput{
		Action:     audit.ActionShareMarkdownSave,
		TargetType: "document",
		TargetID:   response.Document.ID,
		Metadata: map[string]any{
			"sizeBytes": response.Document.SizeBytes,
		},
	})
	api.WriteJSON(w, http.StatusOK, response)
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidInput):
		api.WriteError(w, http.StatusBadRequest, "invalid share input")
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "share link not found")
	case errors.Is(err, document.ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrExpired):
		api.WriteError(w, http.StatusForbidden, "share link expired")
	case errors.Is(err, ErrDisabled):
		api.WriteError(w, http.StatusForbidden, "share link disabled")
	case errors.Is(err, ErrUnsupportedFile):
		api.WriteError(w, http.StatusBadRequest, "unsupported shared file")
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
