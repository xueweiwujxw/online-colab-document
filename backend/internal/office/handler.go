package office

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

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

var collabUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// SheetsWebSocket is a security boundary, not a replacement collaboration
// implementation. It authenticates a native Casual client, re-checks the
// current project permission, then proxies the binary Yjs protocol to the
// official Hocuspocus service with a server-selected role.
func (h Handler) SheetsWebSocket(w http.ResponseWriter, r *http.Request) {
	h.proxyCollab(w, r, "sheets", h.service.cfg.SheetsInternalWSURL)
}

func (h Handler) DocsWebSocket(w http.ResponseWriter, r *http.Request) {
	h.proxyCollab(w, r, "docs", h.service.cfg.DocsInternalWSURL)
}

func (h Handler) proxyCollab(w http.ResponseWriter, r *http.Request, kind, upstream string) {
	room := r.URL.Query().Get("room")
	token := r.URL.Query().Get("share")
	if room == "" || token == "" || upstream == "" {
		api.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	role, err := h.service.AuthorizeCollab(r.Context(), token, room, kind)
	if err != nil {
		h.writeError(w, "casual websocket authorization failed", err)
		return
	}
	target, err := url.Parse(upstream)
	if err != nil {
		h.logger.Error("casual websocket URL invalid", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	query := target.Query()
	query.Set("room", room)
	query.Set("role", role)
	target.RawQuery = query.Encode()
	upstreamConn, _, err := websocket.DefaultDialer.DialContext(r.Context(), target.String(), nil)
	if err != nil {
		h.logger.Error("casual websocket upstream failed", "error", err)
		api.WriteError(w, http.StatusBadGateway, "collaboration service unavailable")
		return
	}
	defer upstreamConn.Close()
	clientConn, err := collabUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()
	done := make(chan struct{}, 2)
	copyFrames := func(dst, src *websocket.Conn) {
		defer func() { done <- struct{}{} }()
		for {
			messageType, payload, readErr := src.ReadMessage()
			if readErr != nil {
				return
			}
			if writeErr := dst.WriteMessage(messageType, payload); writeErr != nil {
				return
			}
		}
	}
	go copyFrames(upstreamConn, clientConn)
	go copyFrames(clientConn, upstreamConn)
	<-done
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

func (h Handler) CollabSession(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	session, err := h.service.CollabSession(r.Context(), currentUser, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "office collab session failed", err)
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

func (h Handler) WOPIInfo(w http.ResponseWriter, r *http.Request) {
	doc, canEdit, version, err := h.service.WOPIInfo(r.Context(), r.URL.Query().Get("access_token"), r.PathValue("id"))
	if err != nil {
		h.writeError(w, "wopi info failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"BaseFileName": doc.OriginalFilename,
		"Size":         doc.SizeBytes,
		"Version":      version,
		"UserCanWrite": canEdit,
		"ReadOnly":     !canEdit,
	})
}

func (h Handler) WOPIContent(w http.ResponseWriter, r *http.Request) {
	doc, reader, version, err := h.service.WOPIContent(r.Context(), r.URL.Query().Get("access_token"), r.PathValue("id"))
	if err != nil {
		h.writeError(w, "wopi content failed", err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", doc.MimeType)
	w.Header().Set("X-WOPI-ItemVersion", version)
	_, _ = io.Copy(w, reader)
}

func (h Handler) WOPISave(w http.ResponseWriter, r *http.Request) {
	doc, size, err := h.service.WOPISave(r.Context(), r.URL.Query().Get("access_token"), r.PathValue("id"), r.Body)
	if err != nil {
		h.writeError(w, "wopi save failed", err)
		return
	}
	h.recordAudit(r, audit.RecordInput{
		Action:     audit.ActionOfficeSave,
		TargetType: "document",
		TargetID:   doc.ID,
		Metadata:   map[string]any{"fileExt": doc.FileExt, "size": size, "via": "wopi"},
	})
	w.Header().Set("X-WOPI-ItemVersion", strconv.FormatInt(time.Now().UnixNano(), 10))
	api.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
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
