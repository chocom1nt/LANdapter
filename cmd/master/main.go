package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/chocom1nt/LANdapter/internal/master"
	"github.com/chocom1nt/LANdapter/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	var cfg common.MasterConfig
	if err := common.LoadConfig("yaml", "configs/master.yaml", &cfg); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger := common.InitLogger(slog.LevelInfo)

	if len(os.Getenv("LANDAPTER_ADMIN_TOKEN")) < 32 {
		logger.Error("LANDAPTER_ADMIN_TOKEN must contain at least 32 characters")
		os.Exit(1)
	}

	var (
		store *storage.PostgresStorage
		err   error
	)

	backoff := time.Second
	const maxBackoff = 10 * time.Second

	for {
		store, err = storage.NewPostgresStorage(cfg.DB)
		if err == nil {
			break
		}

		logger.Error(
			"Failed to connect to PostgreSQL; retrying",
			"host", cfg.DB.Host,
			"port", cfg.DB.Port,
			"database", cfg.DB.DBName,
			"error", err,
			"retry_in", backoff,
		)

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			logger.Info("Shutdown requested while waiting for PostgreSQL")
			return
		case <-timer.C:
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	defer func() {
		if err := store.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		}
	}()

	srv := master.NewServer(&cfg, logger, store)
	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("LANdapter Master started")

	<-ctx.Done()

	logger.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("Shutdown error", "error", err)
	}

	logger.Info("Shutdown complete")
}