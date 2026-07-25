package common

import (
	"log/slog"
	"os"
)

func InitLogger(level slog.Level) *slog.Logger {
    handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    return slog.New(handler)
}