package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                         string
	HTTPAddr                       string
	DatabaseURL                    string
	RedisAddr                      string
	S3Endpoint                     string
	S3AccessKey                    string
	S3SecretKey                    string
	S3Bucket                       string
	S3UseSSL                       bool
	FrontendOrigin                 string
	SessionCookieName              string
	SessionTTLHours                int
	PasswordHashPepper             string
	OIDCEnabled                    bool
	OIDCIssuerURL                  string
	OIDCClientID                   string
	OIDCClientSecret               string
	OIDCRedirectURL                string
	OIDCScopes                     []string
	OIDCAutoMergeByEmail           bool
	DocumentMaxUploadBytes         int64
	OfficeCollabPublicURL          string
	OfficeCollabEnabled            bool
	CasualJWTSecret                string
	PublicAppURL                   string
	PublicAPIURL                   string
	BackendInternalURL             string
	MarkdownSnapshotUpdateInterval int
}

func Load() Config {
	return Config{
		AppEnv:                         getEnv("APP_ENV", "development"),
		HTTPAddr:                       getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:                    getEnv("DATABASE_URL", "postgres://docs:docs@postgres:5432/docs?sslmode=disable"),
		RedisAddr:                      getEnv("REDIS_ADDR", "redis:6379"),
		S3Endpoint:                     getEnv("S3_ENDPOINT", "http://minio:9000"),
		S3AccessKey:                    getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:                    getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:                       getEnv("S3_BUCKET", "docs"),
		S3UseSSL:                       getBoolEnv("S3_USE_SSL", false),
		FrontendOrigin:                 getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		SessionCookieName:              getEnv("SESSION_COOKIE_NAME", "docs_session"),
		SessionTTLHours:                getIntEnv("SESSION_TTL_HOURS", 168),
		PasswordHashPepper:             getEnvAny([]string{"PASSWORD_HASH_PEPPER", "SESSION_SECRET"}, ""),
		OIDCEnabled:                    getBoolEnv("OIDC_ENABLED", false),
		OIDCIssuerURL:                  getEnv("OIDC_ISSUER_URL", ""),
		OIDCClientID:                   getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:               getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:                getEnv("OIDC_REDIRECT_URL", ""),
		OIDCScopes:                     getListEnv("OIDC_SCOPES", []string{"openid", "email", "profile"}),
		OIDCAutoMergeByEmail:           getBoolEnv("OIDC_AUTO_MERGE_BY_EMAIL", false),
		DocumentMaxUploadBytes:         getInt64EnvAny([]string{"DOCUMENT_MAX_UPLOAD_BYTES", "MAX_UPLOAD_BYTES"}, 50<<20),
		OfficeCollabPublicURL:          getEnv("OFFICE_COLLAB_PUBLIC_URL", ""),
		OfficeCollabEnabled:            getBoolEnv("OFFICE_COLLAB_ENABLED", false),
		CasualJWTSecret:                getEnv("CASUAL_JWT_SECRET", ""),
		PublicAppURL:                   getEnv("PUBLIC_APP_URL", "http://localhost:3000"),
		PublicAPIURL:                   getEnv("PUBLIC_API_URL", "http://localhost:8080"),
		BackendInternalURL:             getEnv("BACKEND_INTERNAL_URL", "http://backend:8080"),
		MarkdownSnapshotUpdateInterval: getIntEnv("MARKDOWN_SNAPSHOT_UPDATE_INTERVAL", 100),
	}
}

func (c Config) LogLevel() slog.Level {
	if c.AppEnv == "development" {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAny(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt64Env(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt64EnvAny(keys []string, fallback int64) int64 {
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getListEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
