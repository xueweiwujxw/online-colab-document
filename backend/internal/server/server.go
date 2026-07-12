package server

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"online-colab-document/backend/internal/auth/local"
	oidcauth "online-colab-document/backend/internal/auth/oidc"
	"online-colab-document/backend/internal/config"
	"online-colab-document/backend/internal/document"
	"online-colab-document/backend/internal/health"
	"online-colab-document/backend/internal/middleware"
	"online-colab-document/backend/internal/onlyoffice"
	"online-colab-document/backend/internal/permission"
	"online-colab-document/backend/internal/storage"
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
		authHandler := local.NewHandler(authService, logger, cfg.SessionCookieName, cfg.AppEnv == "production")
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
		)
		mux.HandleFunc("GET /api/auth/oidc/login", oidcHandler.Login)
		mux.HandleFunc("GET /api/auth/oidc/callback", oidcHandler.Callback)

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
			permissionRepo := permission.NewPostgresRepository(db)
			permissionService := permission.NewService(permissionRepo)
			permissionHandler := permission.NewHandler(permissionService, logger)
			documentRepo := document.NewPostgresRepository(db)
			documentService := document.NewService(documentRepo, objectStorage, permissionService, cfg.DocumentMaxUploadBytes)
			documentHandler := document.NewHandler(documentService, logger)
			onlyOfficeService := onlyoffice.NewService(
				onlyoffice.Config{
					Enabled:          cfg.OnlyOfficeEnabled,
					PublicURL:        cfg.OnlyOfficePublicURL,
					JWTSecret:        cfg.OnlyOfficeJWTSecret,
					PublicAPIURL:     cfg.PublicAPIURL,
					CallbackBaseURL:  cfg.BackendInternalURL,
					MaxDownloadBytes: cfg.DocumentMaxUploadBytes,
				},
				documentRepo,
				permissionService,
				objectStorage,
			)
			onlyOfficeHandler := onlyoffice.NewHandler(onlyOfficeService, logger)
			requireAuth := func(next http.HandlerFunc) http.Handler {
				return middleware.RequireAuth(authService, cfg.SessionCookieName, next)
			}
			mux.Handle("GET /api/documents", requireAuth(documentHandler.List))
			mux.Handle("POST /api/documents/upload", requireAuth(documentHandler.Upload))
			mux.Handle("GET /api/documents/{id}", requireAuth(documentHandler.Get))
			mux.Handle("GET /api/documents/{id}/download", requireAuth(documentHandler.Download))
			mux.Handle("DELETE /api/documents/{id}", requireAuth(documentHandler.Delete))
			mux.Handle("GET /api/documents/{id}/versions", requireAuth(documentHandler.Versions))
			mux.Handle("GET /api/documents/{id}/permissions", requireAuth(permissionHandler.List))
			mux.Handle("POST /api/documents/{id}/permissions", requireAuth(permissionHandler.Grant))
			mux.Handle("DELETE /api/documents/{id}/permissions/{permissionId}", requireAuth(permissionHandler.Delete))
			mux.Handle("GET /api/documents/{id}/onlyoffice/config", requireAuth(onlyOfficeHandler.Config))
			mux.HandleFunc("POST /api/onlyoffice/callback/{documentId}", onlyOfficeHandler.Callback)
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
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func withCORS(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.FrontendOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", cfg.FrontendOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
