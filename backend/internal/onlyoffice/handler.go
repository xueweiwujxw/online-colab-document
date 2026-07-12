package onlyoffice

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/middleware"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) Handler {
	return Handler{service: service, logger: logger}
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
	default:
		h.logger.Error(logMessage, "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
