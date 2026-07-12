package document

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

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
	docs, err := h.service.List(r.Context(), currentUser.ID)
	if err != nil {
		h.writeError(w, "list documents failed", err)
		return
	}
	items := make([]PublicDocument, 0, len(docs))
	for _, doc := range docs {
		canManage, err := h.service.CanManage(r.Context(), currentUser.ID, doc.ID)
		if err != nil {
			h.writeError(w, "check document manage permission failed", err)
			return
		}
		items = append(items, ToPublic(doc, canManage))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.service.MaxUploadBytes()+1024)
	if err := r.ParseMultipartForm(h.service.MaxUploadBytes()); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	doc, err := h.service.Upload(r.Context(), UploadInput{
		OwnerID:          currentUser.ID,
		OriginalFilename: header.Filename,
		HeaderMimeType:   header.Header.Get("Content-Type"),
		SizeBytes:        header.Size,
		Reader:           file,
	})
	if err != nil {
		h.writeError(w, "upload document failed", err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, ToPublic(doc, true))
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	doc, err := h.service.Get(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "get document failed", err)
		return
	}
	canManage, err := h.service.CanManage(r.Context(), currentUser.ID, doc.ID)
	if err != nil {
		h.writeError(w, "check document manage permission failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, ToPublic(doc, canManage))
}

func (h Handler) Download(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	download, err := h.service.Download(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "download document failed", err)
		return
	}
	defer download.Reader.Close()

	w.Header().Set("Content-Type", download.Document.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", download.Document.SizeBytes))
	w.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": download.Document.OriginalFilename}),
	)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, download.Reader); err != nil {
		h.logger.Error("stream document failed", "error", err)
	}
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if err := h.service.Delete(r.Context(), currentUser.ID, r.PathValue("id")); err != nil {
		h.writeError(w, "delete document failed", err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) Versions(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := middleware.CurrentUser(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	versions, err := h.service.Versions(r.Context(), currentUser.ID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, "list document versions failed", err)
		return
	}
	items := make([]PublicVersion, 0, len(versions))
	for _, version := range versions {
		items = append(items, VersionToPublic(version))
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) writeError(w http.ResponseWriter, logMessage string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		api.WriteError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, ErrNotFound):
		api.WriteError(w, http.StatusNotFound, "document not found")
	case errors.Is(err, ErrInvalidFile):
		api.WriteError(w, http.StatusBadRequest, "invalid file")
	case errors.Is(err, ErrFileTooLarge):
		api.WriteError(w, http.StatusRequestEntityTooLarge, "file too large")
	case errors.Is(err, ErrUnsupportedType):
		api.WriteError(w, http.StatusBadRequest, "unsupported file type")
	default:
		h.logger.Error(logMessage, "error", err)
		api.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
