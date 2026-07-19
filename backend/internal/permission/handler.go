package permission

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	permissions, err := h.service.List(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "list permissions failed", err)
		return
	}
	items := make([]PublicPermission, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, ToPublic(permission))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) Grant(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Permission  string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	permission, err := h.service.Grant(r.Context(), GrantInput{
		ActorID:     currentUser.ID,
		DocumentID:  r.PathValue("id"),
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		Permission:  req.Permission,
	})
	if err != nil {
		h.writeError(w, "grant permission failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionPermissionGrant,
		TargetType:  "document",
		TargetID:    permission.DocumentID,
		Metadata: map[string]any{
			"permissionId": permission.ID,
			"subjectType":  permission.SubjectType,
			"subjectId":    permission.SubjectID,
			"permission":   permission.Permission,
		},
	})
	api.WriteJSON(w, http.StatusCreated, ToPublic(permission))
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	err := h.service.Delete(r.Context(), currentUser.ID, r.PathValue("id"), r.PathValue("permissionId"))
	if err != nil {
		h.writeError(w, "delete permission failed", err)
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionPermissionDelete,
		TargetType:  "document",
		TargetID:    r.PathValue("id"),
		Metadata: map[string]any{
			"permissionId": r.PathValue("permissionId"),
		},
	})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrInvalidInput):
		api.WriteError(w, http.StatusBadRequest, "invalid permission input")
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "permission not found")
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
