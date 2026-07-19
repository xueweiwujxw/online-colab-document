package office

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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

func (h Handler) Session(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	session, err := h.service.Session(r.Context(), currentUser, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "office session failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, session)
}

func (h Handler) Save(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	doc, size, err := h.service.Save(r.Context(), currentUser, r.PathValue("id"), r.Body)
	if err != nil {
		h.writeError(w, "office save failed", err)
		return
	}
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &currentUser.ID,
		Action:      audit.ActionOfficeSave,
		TargetType:  "document",
		TargetID:    doc.ID,
		Metadata: map[string]any{
			"fileExt": doc.FileExt,
			"size":    size,
		},
	})
	api.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "etag": doc.ID})
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, document.ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrUnsupportedFile):
		api.WriteError(w, http.StatusBadRequest, "unsupported office document")
	case errors.Is(err, ErrTooLarge):
		api.WriteError(w, http.StatusRequestEntityTooLarge, "office document too large")
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
