package health

import (
	"context"
	"net"
	"net/url"
	"time"

	"online-colab-document/backend/internal/config"
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

	parsed, err := url.Parse(c.cfg.S3Endpoint)
	if err != nil || parsed.Host == "" {
		return "error"
	}

	return c.checkTCP(ctx, parsed.Host)
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
