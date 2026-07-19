package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"online-colab-document/backend/internal/config"
	dbmigrate "online-colab-document/backend/internal/db"
	"online-colab-document/backend/internal/server"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel(),
	}))

	database, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error("database open failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelMigration()
	if err := database.PingContext(migrationCtx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	if err := dbmigrate.Migrate(migrationCtx, database); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	app := server.New(cfg, logger, database)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		errCh <- app.Start()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("server stopped")
	case err := <-errCh:
		if err != nil {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}
}
