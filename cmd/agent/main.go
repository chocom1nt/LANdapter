package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chocom1nt/LANdapter/internal/agent"
	"github.com/chocom1nt/LANdapter/internal/common"
)

func main() {
    var cfg common.AgentConfig
    if err := common.LoadConfig("yaml", "configs/agent.yaml", &cfg); err != nil {
        slog.Error("Failed to load config", "error", err)
        os.Exit(1)
    }

    logger := common.InitLogger(slog.LevelInfo)

    ag, err := agent.NewAgent(&cfg, logger)
    if err != nil {
        logger.Error("Failed to create agent", "error", err)
        os.Exit(1)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        if err := ag.Run(ctx); err != nil && err != context.Canceled {
            logger.Error("Agent run error", "error", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down agent...")
    cancel()
    time.Sleep(2 * time.Second)
    logger.Info("Agent stopped")
}