package onlyoffice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
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

func (h Handler) Config(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	cfg, err := h.service.Config(r.Context(), currentUser, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "onlyoffice config failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, cfg)
}

func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	download, err := h.service.Download(r.Context(), r.PathValue("documentId"), r.URL.Query().Get("token"))
	if err != nil {
		h.writeError(w, "onlyoffice download failed", err)
		return
	}
	defer download.Reader.Close()
	w.Header().Set("Content-Type", download.Document.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.Document.SizeBytes))
	w.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": download.Document.OriginalFilename}),
	)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, download.Reader); err != nil {
		h.logger.Error("stream onlyoffice document failed", "error", err)
	}
}

func (h Handler) Callback(w http.ResponseWriter, r *http.Request) {
	var req CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteJSON(w, http.StatusOK, CallbackResponse{Error: 1})
		return
	}
	resp, err := h.service.Callback(r.Context(), r.PathValue("documentId"), req, r.Header.Get("Authorization"))
	if err != nil {
		h.logger.Error("onlyoffice callback failed", "error", err)
		api.WriteJSON(w, http.StatusOK, CallbackResponse{Error: 1})
		return
	}
	if resp.Error == 0 && (req.Status == 2 || req.Status == 6) {
		h.recordAudit(r, audit.RecordInput{
			Action:     audit.ActionOnlyOfficeSave,
			TargetType: "document",
			TargetID:   r.PathValue("documentId"),
			Metadata: map[string]any{
				"status": req.Status,
			},
		})
	}
	api.WriteJSON(w, http.StatusOK, resp)
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrDisabled):
		api.WriteError(w, http.StatusNotFound, "onlyoffice disabled")
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, document.ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrUnsupportedFile):
		api.WriteError(w, http.StatusBadRequest, "unsupported office document")
	case errors.Is(err, ErrInvalidToken):
		api.WriteError(w, http.StatusUnauthorized, "invalid onlyoffice token")
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
