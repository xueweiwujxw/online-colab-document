package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/storage"
)

type Handler struct {
	storage      storage.Storage
	adminStorage storage.AdminStorage
	audit        auditRecorder
	logger       *slog.Logger
}
type auditRecorder interface {
	Record(context.Context, audit.RecordInput) error
}

func NewHandler(objectStorage storage.Storage, adminStorage storage.AdminStorage, recorder auditRecorder, logger *slog.Logger) Handler {
	return Handler{storage: objectStorage, adminStorage: adminStorage, audit: recorder, logger: logger}
}

func (h Handler) Storage(w http.ResponseWriter, r *http.Request) {
	usage, err := h.adminStorage.Usage(r.Context())
	if err != nil {
		h.logger.Error("storage usage failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "storage unavailable")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.adminStorage.ListObjects(r.Context(), r.URL.Query().Get("prefix"), limit)
	if err != nil {
		h.logger.Error("storage list failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "storage unavailable")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"usage": usage, "items": items})
}

func (h Handler) DeleteObject(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		api.WriteError(w, http.StatusBadRequest, "invalid object key")
		return
	}
	if err := h.storage.DeleteObject(r.Context(), key); err != nil {
		if errors.Is(err, context.Canceled) {
			api.WriteError(w, http.StatusRequestTimeout, "request cancelled")
		} else {
			h.logger.Error("storage delete failed", "error", err)
			api.WriteError(w, http.StatusInternalServerError, "storage delete failed")
		}
		return
	}
	actorID := currentUser.ID
	if h.audit != nil {
		_ = h.audit.Record(r.Context(), audit.RecordInput{ActorUserID: &actorID, Action: "admin.storage_delete", TargetType: "storage_object", TargetID: key})
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
