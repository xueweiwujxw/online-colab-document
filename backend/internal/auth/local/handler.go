package local

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/user"
)

type Handler struct {
	service    *Service
	logger     *slog.Logger
	cookieName string
	secure     bool
	audit      AuditRecorder
}

type AuditRecorder interface {
	Record(ctx context.Context, input audit.RecordInput) error
}

func NewHandler(service *Service, logger *slog.Logger, cookieName string, secure bool) Handler {
	return Handler{
		service:    service,
		logger:     logger,
		cookieName: cookieName,
		secure:     secure,
	}
}

func (h Handler) WithAudit(recorder AuditRecorder) Handler {
	h.audit = recorder
	return h
}

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := h.service.Register(r.Context(), RegisterInput{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			api.WriteError(w, http.StatusBadRequest, "invalid registration input")
		case errors.Is(err, ErrEmailAlreadyUsed):
			api.WriteError(w, http.StatusConflict, "email already registered")
		default:
			h.logger.Error("register failed", "error", err)
			api.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	api.WriteJSON(w, http.StatusCreated, user.ToPublic(u))
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	authSession, err := h.service.Login(r.Context(), LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			api.WriteError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		h.logger.Error("login failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	actorUserID := authSession.User.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionLogin,
		TargetType:  "user",
		TargetID:    authSession.User.ID,
	})
	http.SetCookie(w, session.Cookie(h.cookieName, authSession.Token, authSession.ExpiresAt, h.secure))
	api.WriteJSON(w, http.StatusOK, user.ToPublic(authSession.User))
}

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.tokenFromCookie(r)
	var actorUserID *string
	if u, err := h.service.CurrentUser(r.Context(), token); err == nil {
		id := u.ID
		actorUserID = &id
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.logger.Error("logout failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if actorUserID != nil {
		h.recordAudit(r, audit.RecordInput{
			ActorUserID: actorUserID,
			Action:      audit.ActionLogout,
			TargetType:  "user",
			TargetID:    *actorUserID,
		})
	}
	http.SetCookie(w, session.ExpiredCookie(h.cookieName, h.secure))
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.service.CurrentUser(r.Context(), h.tokenFromCookie(r))
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		h.logger.Error("current user failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	api.WriteJSON(w, http.StatusOK, user.ToPublic(u))
}

func (h Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	err := h.service.ChangePassword(r.Context(), ChangePasswordInput{
		UserID:          currentUser.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			api.WriteError(w, http.StatusBadRequest, "invalid password input")
		case errors.Is(err, ErrInvalidCredentials):
			api.WriteError(w, http.StatusUnauthorized, "invalid current password")
		case errors.Is(err, ErrPasswordUnsupported):
			api.WriteError(w, http.StatusBadRequest, "password change unsupported")
		case errors.Is(err, ErrUnauthenticated):
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		default:
			h.logger.Error("change password failed", "error", err)
			api.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionPasswordChange,
		TargetType:  "user",
		TargetID:    currentUser.ID,
	})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	updatedUser, err := h.service.UpdateProfile(r.Context(), UpdateProfileInput{
		UserID:      currentUser.ID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			api.WriteError(w, http.StatusBadRequest, "invalid profile input")
		case errors.Is(err, ErrUnauthenticated):
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		default:
			h.logger.Error("update profile failed", "error", err)
			api.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionProfileUpdate,
		TargetType:  "user",
		TargetID:    currentUser.ID,
	})
	api.WriteJSON(w, http.StatusOK, user.ToPublic(updatedUser))
}

func (h Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	items, err := h.service.ListSessions(r.Context(), currentUser.ID, h.tokenFromCookie(r))
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		h.logger.Error("list sessions failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string][]PublicSession{"items": items})
}

func (h Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	sessionID := r.PathValue("id")
	err := h.service.RevokeSession(r.Context(), currentUser.ID, sessionID, h.tokenFromCookie(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			api.WriteError(w, http.StatusBadRequest, "invalid session")
		case errors.Is(err, ErrCurrentSession):
			api.WriteError(w, http.StatusBadRequest, "cannot revoke current session")
		case errors.Is(err, ErrSessionForbidden):
			api.WriteError(w, http.StatusNotFound, "session not found")
		default:
			h.logger.Error("revoke session failed", "error", err)
			api.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	actorUserID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionSessionRevoke,
		TargetType:  "session",
		TargetID:    sessionID,
	})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) tokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(h.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
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
