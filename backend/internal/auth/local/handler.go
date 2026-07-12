package local

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

type Handler struct {
	service    *Service
	logger     *slog.Logger
	cookieName string
	secure     bool
}

func NewHandler(service *Service, logger *slog.Logger, cookieName string, secure bool) Handler {
	return Handler{
		service:    service,
		logger:     logger,
		cookieName: cookieName,
		secure:     secure,
	}
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
	http.SetCookie(w, session.Cookie(h.cookieName, authSession.Token, authSession.ExpiresAt, h.secure))
	api.WriteJSON(w, http.StatusOK, user.ToPublic(authSession.User))
}

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.tokenFromCookie(r)
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.logger.Error("logout failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
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

func (h Handler) tokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(h.cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
