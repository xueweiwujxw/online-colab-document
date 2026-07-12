package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	DatabaseURL        string
	RedisAddr          string
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3UseSSL           bool
	FrontendOrigin     string
	SessionCookieName  string
	SessionTTLHours    int
	PasswordHashPepper string
}

func Load() Config {
	return Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://docs:docs@postgres:5432/docs?sslmode=disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "redis:6379"),
		S3Endpoint:         getEnv("S3_ENDPOINT", "http://minio:9000"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:           getEnv("S3_BUCKET", "docs"),
		S3UseSSL:           getBoolEnv("S3_USE_SSL", false),
		FrontendOrigin:     getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "docs_session"),
		SessionTTLHours:    getIntEnv("SESSION_TTL_HOURS", 168),
		PasswordHashPepper: getEnv("PASSWORD_HASH_PEPPER", ""),
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
