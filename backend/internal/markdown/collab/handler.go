package collab

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/user"
)

type Handler struct {
	service       *Service
	authenticator middleware.Authenticator
	cookieName    string
	allowedOrigin string
	logger        *slog.Logger
	upgrader      websocket.Upgrader
}

func NewHandler(service *Service, authenticator middleware.Authenticator, cookieName string, allowedOrigin string, logger *slog.Logger) Handler {
	return Handler{
		service:       service,
		authenticator: authenticator,
		cookieName:    cookieName,
		allowedOrigin: allowedOrigin,
		logger:        logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if allowedOrigin == "" {
					return true
				}
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				parsed, err := url.Parse(origin)
				if err != nil {
					return false
				}
				allowed, err := url.Parse(allowedOrigin)
				if err != nil {
					return false
				}
				return parsed.Scheme == allowed.Scheme && parsed.Host == allowed.Host
			},
		},
	}
}

func (h Handler) Snapshot(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	snapshot, err := h.service.Snapshot(r.Context(), currentUser, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "get markdown snapshot failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, snapshot)
}

func (h Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.currentUser(r)
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	session, err := h.service.Join(r.Context(), currentUser, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "join markdown websocket failed", err)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.service.Leave(session)
		h.logger.Error("upgrade markdown websocket failed", "error", err)
		return
	}
	defer conn.Close()
	defer h.service.Leave(session)

	done := make(chan struct{})
	go h.writeLoop(conn, session, done)
	h.readLoop(conn, session)
	close(done)
}

func (h Handler) currentUser(r *http.Request) (user.User, bool) {
	cookie, err := r.Cookie(h.cookieName)
	if err != nil {
		return user.User{}, false
	}
	currentUser, err := h.authenticator.CurrentUser(r.Context(), cookie.Value)
	if err != nil {
		return user.User{}, false
	}
	return currentUser, true
}

func (h Handler) readLoop(conn *websocket.Conn, session *ClientSession) {
	conn.SetReadLimit(2 << 20)
	for {
		var update ClientUpdate
		if err := conn.ReadJSON(&update); err != nil {
			return
		}
		if err := h.service.ApplyClientMessage(context.Background(), session, update); err != nil && !errors.Is(err, ErrForbidden) {
			h.logger.Error("apply markdown websocket message failed", "error", err)
			session.enqueue(ServerMessage{Type: "error", Error: "failed to apply update"})
		}
	}
}

func (h Handler) writeLoop(conn *websocket.Conn, session *ClientSession, done <-chan struct{}) {
	for {
		select {
		case message, ok := <-session.Receive:
			if !ok {
				return
			}
			if err := conn.WriteJSON(message); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, document.ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrUnsupportedFile):
		api.WriteError(w, http.StatusBadRequest, "unsupported markdown document")
	default:
		h.logger.Error(logMessage, "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
