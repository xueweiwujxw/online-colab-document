package audit

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/middleware"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) Handler {
	return Handler{service: service, logger: logger}
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !currentUser.IsAdmin {
		api.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	filter, err := parseListFilter(r)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid audit log filter")
		return
	}
	requestedLimit := filter.Limit
	if requestedLimit <= 0 || requestedLimit > 100 {
		requestedLimit = 50
	}
	filter.Limit = requestedLimit + 1
	logs, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list audit logs failed", "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	hasMore := len(logs) > requestedLimit
	if hasMore {
		logs = logs[:requestedLimit]
	}
	items := make([]PublicLog, 0, len(logs))
	for _, log := range logs {
		items = append(items, ToPublic(log))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "hasMore": hasMore})
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	query := r.URL.Query()
	filter := ListFilter{
		Action:     query.Get("action"),
		TargetType: query.Get("targetType"),
		TargetID:   query.Get("targetId"),
		IPAddr:     query.Get("ipAddr"),
	}
	if actorUserID := query.Get("actorUserId"); actorUserID != "" {
		filter.ActorUserID = &actorUserID
	}
	var err error
	filter.From, err = parseOptionalTime(query.Get("from"))
	if err != nil {
		return ListFilter{}, err
	}
	filter.To, err = parseOptionalTime(query.Get("to"))
	if err != nil {
		return ListFilter{}, err
	}
	filter.Limit, err = parseOptionalInt(query.Get("limit"))
	if err != nil {
		return ListFilter{}, err
	}
	filter.Offset, err = parseOptionalInt(query.Get("offset"))
	if err != nil {
		return ListFilter{}, err
	}
	return filter, nil
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, errors.New("negative integer")
	}
	return parsed, nil
}
