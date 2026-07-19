package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/auth/local"
	"online-colab-document/backend/internal/auth/session"
)

const (
	stateCookieName = "docs_oidc_state"
	nonceCookieName = "docs_oidc_nonce"
)

type Handler struct {
	service      *Service
	logger       *slog.Logger
	cookieName   string
	secure       bool
	sessionTTL   time.Duration
	frontendURL  string
	transientTTL time.Duration
	audit        AuditRecorder
}

type AuditRecorder interface {
	Record(ctx context.Context, input audit.RecordInput) error
}

func NewHandler(service *Service, logger *slog.Logger, cookieName string, secure bool, sessionTTL time.Duration, frontendURL string) Handler {
	return Handler{
		service:      service,
		logger:       logger,
		cookieName:   cookieName,
		secure:       secure,
		sessionTTL:   sessionTTL,
		frontendURL:  frontendURL,
		transientTTL: 10 * time.Minute,
	}
}

func (h Handler) WithAudit(recorder AuditRecorder) Handler {
	h.audit = recorder
	return h
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := NewStateToken()
	if err != nil {
		h.logger.Error("oidc state generation failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	nonce, err := NewNonce()
	if err != nil {
		h.logger.Error("oidc nonce generation failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	loginURL, err := h.service.LoginURL(r.Context(), state, nonce)
	if err != nil {
		h.writeAuthError(w, "oidc login failed", err)
		return
	}
	expires := time.Now().Add(h.transientTTL)
	http.SetCookie(w, transientCookie(stateCookieName, state, expires, h.secure))
	http.SetCookie(w, transientCookie(nonceCookieName, nonce, expires, h.secure))
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (h Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if errText := r.URL.Query().Get("error"); errText != "" {
		api.WriteError(w, http.StatusUnauthorized, "oidc login failed")
		return
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	expectedState := cookieValue(r, stateCookieName)
	nonce := cookieValue(r, nonceCookieName)
	http.SetCookie(w, expiredTransientCookie(stateCookieName, h.secure))
	http.SetCookie(w, expiredTransientCookie(nonceCookieName, h.secure))
	if state == "" || expectedState == "" || state != expectedState {
		api.WriteError(w, http.StatusBadRequest, "invalid oidc state")
		return
	}
	authSession, err := h.service.Callback(r.Context(), code, nonce, h.sessionTTL)
	if err != nil {
		h.writeAuthError(w, "oidc callback failed", err)
		return
	}
	actorUserID := authSession.User.ID
	h.recordAudit(r, audit.RecordInput{
		ActorUserID: &actorUserID,
		Action:      audit.ActionLogin,
		TargetType:  "user",
		TargetID:    authSession.User.ID,
		Metadata: map[string]any{
			"authSource": "oidc",
		},
	})
	http.SetCookie(w, session.Cookie(h.cookieName, authSession.Token, authSession.ExpiresAt, h.secure))
	http.Redirect(w, r, h.redirectURL(), http.StatusFound)
}

func (h Handler) writeAuthError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrDisabled):
		api.WriteError(w, http.StatusNotFound, "oidc disabled")
	case errors.Is(err, ErrInvalidState):
		api.WriteError(w, http.StatusBadRequest, "invalid oidc state")
	case errors.Is(err, ErrInvalidToken), errors.Is(err, local.ErrInvalidCredentials):
		api.WriteError(w, http.StatusUnauthorized, "oidc login failed")
	case errors.Is(err, ErrEmailConflict):
		api.WriteError(w, http.StatusConflict, "email already registered")
	default:
		h.logger.Error(logMessage, "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h Handler) redirectURL() string {
	if h.frontendURL == "" {
		return "/documents"
	}
	parsed, err := url.Parse(h.frontendURL)
	if err != nil {
		return "/documents"
	}
	parsed.Path = "/documents"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func transientCookie(name string, value string, expires time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/api/auth/oidc",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredTransientCookie(name string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api/auth/oidc",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
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
