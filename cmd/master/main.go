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
    var cfg common.MasterConfig
    if err := common.LoadConfig("yaml", "configs/master.yaml", &cfg); err != nil {
        slog.Error("Failed to load config", "error", err)
        os.Exit(1)
    }

    logger := common.InitLogger(slog.LevelInfo)

    store, err := storage.NewPostgresStorage(cfg.DB)
    if err != nil {
        logger.Error("Failed to connect to DB", "error", err)
        os.Exit(1)
    }
    defer store.Close()

    srv := master.NewServer(&cfg, logger, store)
    if err := srv.Start(); err != nil {
        logger.Error("Failed to start server", "error", err)
        os.Exit(1)
    }

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Stop(ctx); err != nil {
        logger.Error("Shutdown error", "error", err)
    }
    logger.Info("Shutdown complete")
}