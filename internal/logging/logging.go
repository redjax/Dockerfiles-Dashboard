package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	File       string
	Level      slog.Level
	MaxSize    int
	MaxBackups int
	MaxAge     int
}

func Setup(cfg Config) (*slog.Logger, func() error, error) {
	var writers []io.Writer

	// Always log to stdout.
	writers = append(writers, os.Stdout)

	var fileWriter *lumberjack.Logger

	if cfg.File != "" {
		dir := filepath.Dir(cfg.File)

		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, fmt.Errorf(
				"creating log directory: %w",
				err,
			)
		}

		fileWriter = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   true,
		}

		writers = append(writers, fileWriter)
	}

	writer := io.MultiWriter(writers...)

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: cfg.Level,
	})

	logger := slog.New(handler)

	closeFn := func() error {
		if fileWriter != nil {
			return fileWriter.Close()
		}

		return nil
	}

	return logger, closeFn, nil
}

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf(
			"invalid log level %q (expected debug, info, warn, or error)",
			value,
		)
	}
}
