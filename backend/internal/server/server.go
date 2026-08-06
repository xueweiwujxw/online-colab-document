package server

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"online-colab-document/backend/internal/admin"
	"online-colab-document/backend/internal/api"
	"online-colab-document/backend/internal/audit"
	"online-colab-document/backend/internal/auth/local"
	oidcauth "online-colab-document/backend/internal/auth/oidc"
	"online-colab-document/backend/internal/config"
	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/health"
	markdowncollab "online-colab-document/backend/internal/markdown/collab"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/office"
	"online-colab-document/backend/internal/permission"
	"online-colab-document/backend/internal/share"
	"online-colab-document/backend/internal/storage"
	appuser "online-colab-document/backend/internal/user"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger, db *sql.DB) *Server {
	mux := http.NewServeMux()
	healthHandler := health.NewHandler(health.NewChecker(cfg))

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /readyz", healthHandler.Readyz)
	if db != nil {
		authRepo := local.NewPostgresRepository(db)
		authService := local.NewService(
			authRepo,
			authRepo,
			cfg.PasswordHashPepper,
			time.Duration(cfg.SessionTTLHours)*time.Hour,
		)
		auditService := audit.NewService(audit.NewPostgresRepository(db))
		auditHandler := audit.NewHandler(auditService, logger)
		authHandler := local.NewHandler(authService, logger, cfg.SessionCookieName, cfg.AppEnv == "production").WithAudit(auditService)
		mux.HandleFunc("POST /api/auth/local/register", authHandler.Register)
		mux.HandleFunc("POST /api/auth/local/login", authHandler.Login)
		mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
		mux.HandleFunc("GET /api/auth/me", authHandler.Me)

		oidcService := oidcauth.NewService(
			oidcauth.Config{
				Enabled:          cfg.OIDCEnabled,
				IssuerURL:        cfg.OIDCIssuerURL,
				ClientID:         cfg.OIDCClientID,
				ClientSecret:     cfg.OIDCClientSecret,
				RedirectURL:      cfg.OIDCRedirectURL,
				Scopes:           cfg.OIDCScopes,
				AutoMergeByEmail: cfg.OIDCAutoMergeByEmail,
			},
			authRepo,
			authRepo,
			oidcauth.NewProvider,
		)
		oidcHandler := oidcauth.NewHandler(
			oidcService,
			logger,
			cfg.SessionCookieName,
			cfg.AppEnv == "production",
			time.Duration(cfg.SessionTTLHours)*time.Hour,
			cfg.FrontendOrigin,
		).WithAudit(auditService)
		mux.HandleFunc("GET /api/auth/oidc/login", oidcHandler.Login)
		mux.HandleFunc("GET /api/auth/oidc/callback", oidcHandler.Callback)
		requireAuth := func(next http.HandlerFunc) http.Handler {
			return middleware.RequireAuth(authService, cfg.SessionCookieName, next)
		}
		requireAdmin := func(next http.HandlerFunc) http.Handler {
			return requireAuth(func(w http.ResponseWriter, r *http.Request) {
				currentUser, ok := middleware.CurrentUser(r.Context())
				if !ok {
					api.WriteError(w, http.StatusUnauthorized, "unauthenticated")
					return
				}
				if !currentUser.IsAdmin {
					api.WriteError(w, http.StatusForbidden, "forbidden")
					return
				}
				next(w, r)
			})
		}
		mux.Handle("PUT /api/auth/password", requireAuth(authHandler.ChangePassword))
		mux.Handle("PUT /api/auth/profile", requireAuth(authHandler.UpdateProfile))
		mux.Handle("GET /api/auth/sessions", requireAuth(authHandler.ListSessions))
		mux.Handle("DELETE /api/auth/sessions/{id}", requireAuth(authHandler.RevokeSession))
		userService := appuser.NewService(appuser.NewPostgresRepository(db))
		userHandler := appuser.NewHandler(userService, logger)
		mux.Handle("GET /api/users", requireAuth(userHandler.Search))
		mux.Handle("GET /api/admin/users", requireAdmin(userHandler.ListAdmin))
		mux.Handle("PUT /api/admin/users/{id}/password", requireAdmin(authHandler.AdminResetPassword))
		mux.Handle("GET /api/admin/audit-logs", requireAuth(auditHandler.List))

		objectStorage, err := storage.NewMinIOStorage(storage.MinIOConfig{
			Endpoint:  cfg.S3Endpoint,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Bucket:    cfg.S3Bucket,
			UseSSL:    cfg.S3UseSSL,
		})
		if err != nil {
			logger.Error("storage setup failed", "error", err)
		} else {
			adminHandler := admin.NewHandler(objectStorage, objectStorage, auditService, logger)
			mux.Handle("GET /api/admin/storage", requireAdmin(adminHandler.Storage))
			mux.Handle("DELETE /api/admin/storage/object", requireAdmin(adminHandler.DeleteObject))
			authHandler = authHandler.WithAvatarStorage(objectStorage)
			mux.Handle("PUT /api/auth/avatar", requireAuth(authHandler.UpdateAvatar))
			mux.Handle("GET /api/users/{id}/avatar", requireAuth(authHandler.Avatar))
			permissionRepo := permission.NewPostgresRepository(db)
			permissionService := permission.NewService(permissionRepo)
			permissionHandler := permission.NewHandler(permissionService, logger).WithAudit(auditService)
			documentRepo := document.NewPostgresRepository(db)
			documentService := document.NewService(documentRepo, objectStorage, permissionService, cfg.DocumentMaxUploadBytes)
			documentHandler := document.NewHandler(documentService, logger).WithAudit(auditService)
			shareService := share.NewService(
				share.NewPostgresRepository(db),
				documentRepo,
				permissionService,
				objectStorage,
				cfg.PublicAppURL,
			)
			shareHandler := share.NewHandler(shareService, logger).WithAudit(auditService)
			markdownCollabService := markdowncollab.NewService(
				markdowncollab.NewPostgresRepository(db),
				documentRepo,
				permissionService,
				objectStorage,
				cfg.MarkdownSnapshotUpdateInterval,
			)
			markdownCollabHandler := markdowncollab.NewHandler(
				markdownCollabService,
				authService,
				cfg.SessionCookieName,
				cfg.FrontendOrigin,
				logger,
			)
			officeService := office.NewService(
				office.Config{
					Provider:            "casual",
					PublicAPIURL:        cfg.PublicAPIURL,
					CollabPublicURL:     cfg.OfficeCollabPublicURL,
					CollabEnabled:       cfg.OfficeCollabEnabled,
					JWTSecret:           cfg.CasualJWTSecret,
					DocsEditorURL:       cfg.CasualDocsEditorURL,
					SheetsEditorURL:     cfg.CasualSheetsEditorURL,
					SheetsInternalWSURL: cfg.CasualSheetsInternalWSURL,
					DocsInternalWSURL:   cfg.CasualDocsInternalWSURL,
					MaxUploadBytes:      cfg.DocumentMaxUploadBytes,
				},
				documentRepo,
				permissionService,
				objectStorage,
			)
			officeHandler := office.NewHandler(officeService, logger).WithAudit(auditService)
			mux.Handle("GET /api/documents", requireAuth(documentHandler.List))
			mux.Handle("POST /api/documents/upload", requireAuth(documentHandler.Upload))
			mux.Handle("GET /api/documents/{id}", requireAuth(documentHandler.Get))
			mux.Handle("GET /api/documents/{id}/download", requireAuth(documentHandler.Download))
			mux.Handle("GET /api/documents/{id}/markdown", requireAuth(documentHandler.GetMarkdown))
			mux.Handle("PUT /api/documents/{id}/markdown", requireAuth(documentHandler.UpdateMarkdown))
			mux.Handle("GET /api/documents/{id}/markdown/snapshot", requireAuth(markdownCollabHandler.Snapshot))
			mux.HandleFunc("GET /api/documents/{id}/markdown/ws", markdownCollabHandler.WebSocket)
			mux.Handle("DELETE /api/documents/{id}", requireAuth(documentHandler.Delete))
			mux.Handle("GET /api/documents/{id}/versions", requireAuth(documentHandler.Versions))
			mux.Handle("GET /api/documents/{id}/versions/{versionId}/download", requireAuth(documentHandler.DownloadVersion))
			mux.Handle("POST /api/documents/{id}/versions/{versionId}/restore", requireAuth(documentHandler.RestoreVersion))
			mux.Handle("GET /api/documents/{id}/permissions", requireAuth(permissionHandler.List))
			mux.Handle("POST /api/documents/{id}/permissions", requireAuth(permissionHandler.Grant))
			mux.Handle("DELETE /api/documents/{id}/permissions/{permissionId}", requireAuth(permissionHandler.Delete))
			mux.Handle("POST /api/documents/{id}/share-links", requireAuth(shareHandler.Create))
			mux.Handle("GET /api/documents/{id}/share-links", requireAuth(shareHandler.List))
			mux.Handle("DELETE /api/share-links/{id}", requireAuth(shareHandler.Disable))
			mux.HandleFunc("GET /api/share/{token}", shareHandler.Access)
			mux.HandleFunc("GET /api/share/{token}/download", shareHandler.Download)
			mux.HandleFunc("PUT /api/share/{token}/markdown", shareHandler.SaveMarkdown)
			mux.Handle("GET /api/documents/{id}/office/session", requireAuth(officeHandler.Session))
			mux.Handle("GET /api/documents/{id}/office/collab/session", requireAuth(officeHandler.CollabSession))
			mux.Handle("PUT /api/documents/{id}/office/content", requireAuth(officeHandler.Save))
			mux.HandleFunc("GET /wopi/files/{id}", officeHandler.WOPIInfo)
			mux.HandleFunc("GET /wopi/files/{id}/contents", officeHandler.WOPIContent)
			mux.HandleFunc("POST /wopi/files/{id}/contents", officeHandler.WOPISave)
			mux.HandleFunc("GET /casual/sheets/yjs", officeHandler.SheetsWebSocket)
			mux.Handle("GET /casual/sheets/rooms/{id}/info", requireAuth(officeHandler.SheetsRoomInfo))
			mux.Handle("GET /casual/sheets/rooms/{id}/seed", requireAuth(officeHandler.SheetsRoomSeed))
			mux.HandleFunc("GET /casual/docs/yjs", officeHandler.DocsWebSocket)
			mux.Handle("GET /casual/docs/rooms/{id}/seed", requireAuth(officeHandler.DocsRoomSeed))
		}
	}

	handler := withLogging(logger, withCORS(cfg, mux))

	return &Server{
		logger: logger,
		httpServer: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Info(
			"http request",
			"method", r.Method,
			"path", logPath(r.URL.Path),
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func logPath(path string) string {
	if strings.HasPrefix(path, "/api/share/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			parts[3] = "<redacted>"
			return strings.Join(parts, "/")
		}
	}
	return path
}

func withCORS(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.FrontendOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", cfg.FrontendOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}
