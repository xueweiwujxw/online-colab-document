package local

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/storage"
	"online-colab-document/backend/internal/user"
)

type Handler struct {
	service       *Service
	logger        *slog.Logger
	cookieName    string
	secure        bool
	audit         AuditRecorder
	avatarStorage storage.Storage
}

func (h Handler) WithAvatarStorage(objectStorage storage.Storage) Handler {
	h.avatarStorage = objectStorage
	return h
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

func (h Handler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if h.avatarStorage == nil {
		api.WriteError(w, http.StatusServiceUnavailable, "avatar storage unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	bytes, err := io.ReadAll(r.Body)
	if err != nil || len(bytes) == 0 {
		api.WriteError(w, http.StatusBadRequest, "invalid avatar image")
		return
	}
	contentType := http.DetectContentType(bytes)
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[contentType]
	if extension == "" {
		api.WriteError(w, http.StatusBadRequest, "avatar must be a png, jpeg, or webp image")
		return
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		h.logger.Error("avatar id failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	key := filepath.ToSlash(fmt.Sprintf("avatars/%s/%x%s", currentUser.ID, id, extension))
	if err := h.avatarStorage.PutObject(r.Context(), key, bytesReader(bytes), int64(len(bytes)), contentType); err != nil {
		h.logger.Error("avatar upload failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	updated, err := h.service.UpdateAvatar(r.Context(), UpdateAvatarInput{UserID: currentUser.ID, AvatarKey: key})
	if err != nil {
		_ = h.avatarStorage.DeleteObject(r.Context(), key)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if currentUser.AvatarKey != nil {
		_ = h.avatarStorage.DeleteObject(r.Context(), *currentUser.AvatarKey)
	}
	actorID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{ActorUserID: &actorID, Action: audit.ActionAvatarUpdate, TargetType: "user", TargetID: currentUser.ID})
	api.WriteJSON(w, http.StatusOK, user.ToPublic(updated))
}

func (h Handler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !currentUser.IsAdmin {
		api.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req struct {
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	err := h.service.AdminResetPassword(r.Context(), AdminResetPasswordInput{Actor: currentUser, TargetUserID: r.PathValue("id"), NewPassword: req.NewPassword})
	if err != nil {
		if errors.Is(err, ErrPasswordUnsupported) {
			api.WriteError(w, http.StatusBadRequest, "password reset unsupported")
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			api.WriteError(w, http.StatusBadRequest, "invalid password input")
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			api.WriteError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("admin password reset failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	actorID := currentUser.ID
	h.recordAudit(r, audit.RecordInput{ActorUserID: &actorID, Action: "admin.password_reset", TargetType: "user", TargetID: r.PathValue("id")})
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Avatar(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.CurrentUser(r.Context()); !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if h.avatarStorage == nil {
		api.WriteError(w, http.StatusServiceUnavailable, "avatar storage unavailable")
		return
	}
	u, err := h.service.users.FindByID(r.Context(), r.PathValue("id"))
	if err != nil || u.AvatarKey == nil {
		api.WriteError(w, http.StatusNotFound, "avatar not found")
		return
	}
	reader, err := h.avatarStorage.GetObject(r.Context(), *u.AvatarKey)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "avatar not found")
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", contentTypeForAvatar(*u.AvatarKey))
	_, _ = io.Copy(w, reader)
}

func bytesReader(bytes []byte) io.Reader { return &byteReader{bytes: bytes} }

type byteReader struct {
	bytes  []byte
	offset int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.bytes) {
		return 0, io.EOF
	}
	n := copy(p, r.bytes[r.offset:])
	r.offset += n
	return n, nil
}
func contentTypeForAvatar(key string) string {
	switch filepath.Ext(key) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
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
