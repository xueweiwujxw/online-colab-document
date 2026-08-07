package admin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/config"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/storage"
)

type Handler struct {
	storage      storage.Storage
	adminStorage storage.AdminStorage
	audit        auditRecorder
	logger       *slog.Logger
	oidc         OIDCStatus
}
type OIDCStatus struct {
	Enabled          bool     `json:"enabled"`
	IssuerURL        string   `json:"issuerUrl"`
	ClientID         string   `json:"clientId"`
	RedirectURL      string   `json:"redirectUrl"`
	Scopes           []string `json:"scopes"`
	AutoMergeByEmail bool     `json:"autoMergeByEmail"`
	SecretConfigured bool     `json:"secretConfigured"`
	Source           string   `json:"source"`
}
type auditRecorder interface {
	Record(context.Context, audit.RecordInput) error
}

func NewHandler(objectStorage storage.Storage, adminStorage storage.AdminStorage, recorder auditRecorder, logger *slog.Logger, cfg config.Config) Handler {
	return Handler{storage: objectStorage, adminStorage: adminStorage, audit: recorder, logger: logger, oidc: OIDCStatus{Enabled: cfg.OIDCEnabled, IssuerURL: cfg.OIDCIssuerURL, ClientID: cfg.OIDCClientID, RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes, AutoMergeByEmail: cfg.OIDCAutoMergeByEmail, SecretConfigured: cfg.OIDCClientSecret != "", Source: "environment"}}
}

func (h Handler) OIDC(w http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(w, http.StatusOK, h.oidc)
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
	key, ok := validObjectKey(r)
	if !ok {
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

func (h Handler) DownloadObject(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	key, ok := validObjectKey(r)
	if !ok {
		api.WriteError(w, http.StatusBadRequest, "invalid object key")
		return
	}
	reader, err := h.storage.GetObject(r.Context(), key)
	if err != nil {
		h.logger.Error("admin storage download failed", "error", err)
		api.WriteError(w, http.StatusNotFound, "object not found")
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(key)}))
	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("admin storage download stream failed", "error", err)
		return
	}
	actorID := currentUser.ID
	if h.audit != nil {
		_ = h.audit.Record(r.Context(), audit.RecordInput{ActorUserID: &actorID, Action: "admin.storage_download", TargetType: "storage_object", TargetID: key})
	}
}

func validObjectKey(r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	return key, key != "" && !strings.HasPrefix(key, "/") && !strings.Contains(key, "..")
}
