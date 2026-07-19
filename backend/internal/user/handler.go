package user

import (
	"log/slog"
	"net/http"
	"strconv"

	"online-colab-document/backend/internal/api"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) Handler {
	return Handler{service: service, logger: logger}
}

func (h Handler) Search(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.Search(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		h.logger.Error("search users failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
