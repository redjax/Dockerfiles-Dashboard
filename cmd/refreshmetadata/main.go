package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"local/dockerfiles/internal/config"
	"local/dockerfiles/internal/github"
	"local/dockerfiles/internal/logging"
	"local/dockerfiles/internal/services/fsService"
	"local/dockerfiles/internal/services/githubService"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"error:",
			err,
		)

		os.Exit(1)
	}

	level, err := logging.ParseLevel(
		cfg.LogLevel,
	)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"error:",
			err,
		)

		os.Exit(1)
	}

	logger, closeLogger, err := logging.Setup(
		logging.Config{
			File:       cfg.LogFile,
			Level:      level,
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
		},
	)
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"error setting up logging:",
			err,
		)

		os.Exit(1)
	}

	defer closeLogger()

	slog.SetDefault(
		logger,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	fsSvc := fsService.New()

	client := github.NewClient(
		cfg.GithubToken,
		cfg.Workers,
	)

	githubSvc := githubService.New(
		client,
		cfg.GithubUsername,
		cfg.Repository,
		cfg.Workers,
	)

	run := func() {
		if err := refresh(
			cfg,
			fsSvc,
			githubSvc,
		); err != nil {
			slog.Error(
				"metadata refresh failed",
				"error",
				err,
			)
		}
	}

	run()

	if cfg.RefreshInterval == 0 {
		return
	}

	slog.Info(
		"scheduled metadata refresh enabled",
		"interval",
		cfg.RefreshInterval.String(),
	)

	ticker := time.NewTicker(
		cfg.RefreshInterval,
	)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"metadata refresher stopping",
				"reason",
				ctx.Err(),
			)

			return

		case <-ticker.C:
			run()
		}
	}
}

func refresh(
	cfg config.Config,
	fsSvc *fsService.Service,
	githubSvc *githubService.Service,
) error {
	slog.Info(
		"checking packages",
		"user",
		cfg.GithubUsername,
		"repo",
		cfg.Repository,
	)

	// Attempt to load existing metadata, or create initial metadata file on first run.
	oldMetadata, err := fsSvc.LoadJsonMetadata(
		cfg.DataFile,
	)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info(
				"metadata file does not exist; performing initial refresh",
			)

			oldMetadata = fsService.Metadata{}
		} else {
			return fmt.Errorf(
				"loading metadata: %w",
				err,
			)
		}
	}

	packages, err := githubSvc.GetContainers()
	if err != nil {
		return fmt.Errorf(
			"getting packages: %w",
			err,
		)
	}

	slog.Info(
		"found container packages",
		"count",
		len(packages),
	)

	slog.Info(
		"comparing package metadata",
	)

	githubService.Compare(
		oldMetadata.Packages,
		packages,
	)

	repository := fmt.Sprintf(
		"%s/%s",
		cfg.GithubUsername,
		cfg.Repository,
	)

	repositoryURL := fmt.Sprintf(
		"https://github.com/%s/%s",
		cfg.GithubUsername,
		cfg.Repository,
	)

	if err := fsSvc.SaveJsonMetadata(
		cfg.DataFile,
		repository,
		repositoryURL,
		packages,
	); err != nil {
		return fmt.Errorf(
			"saving metadata: %w",
			err,
		)
	}

	slog.Info(
		"wrote packages",
		"count",
		len(packages),
		"file",
		cfg.DataFile,
	)

	return nil
}
