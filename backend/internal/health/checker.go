package health

import (
	"context"
	"net"
	"net/url"
	"time"

	"online-colab-document/backend/internal/config"
	"online-colab-document/backend/internal/storage"
)

type Checker struct {
	cfg     config.Config
	timeout time.Duration
}

func NewChecker(cfg config.Config) Checker {
	return Checker{cfg: cfg, timeout: 2 * time.Second}
}

func (c Checker) Ready(ctx context.Context) map[string]string {
	return map[string]string{
		"database": c.checkPostgres(ctx),
		"redis":    c.checkTCP(ctx, c.cfg.RedisAddr),
		"storage":  c.checkStorage(ctx),
	}
}

func (c Checker) checkPostgres(ctx context.Context) string {
	if c.cfg.DatabaseURL == "" {
		return "skipped"
	}

	parsed, err := url.Parse(c.cfg.DatabaseURL)
	if err != nil || parsed.Host == "" {
		return "error"
	}

	return c.checkTCP(ctx, parsed.Host)
}

func (c Checker) checkStorage(ctx context.Context) string {
	if c.cfg.S3Endpoint == "" {
		return "skipped"
	}

	client, err := storage.NewS3Storage(storage.S3Config{
		Endpoint: c.cfg.S3Endpoint, AccessKey: c.cfg.S3AccessKey,
		SecretKey: c.cfg.S3SecretKey, Bucket: c.cfg.S3Bucket, UseSSL: c.cfg.S3UseSSL,
	})
	if err != nil {
		return "error"
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if err := client.Check(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func (c Checker) checkTCP(ctx context.Context, addr string) string {
	if addr == "" {
		return "skipped"
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "error"
	}
	_ = conn.Close()
	return "ok"
}
