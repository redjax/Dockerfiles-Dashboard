package main

import (
	"fmt"
	"log/slog"
	"os"

	"local/dockerfiles/internal/config"
	"local/dockerfiles/internal/github"
	"local/dockerfiles/internal/logging"
	"local/dockerfiles/internal/services/fsService"
	"local/dockerfiles/internal/services/githubService"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	level, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	logger, closeLogger, err := logging.Setup(logging.Config{
		File:       cfg.LogFile,
		Level:      level,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logging:", err)
		os.Exit(1)
	}
	defer closeLogger()

	slog.SetDefault(logger)

	slog.Info(
		"Checking packages",
		"user", cfg.GithubUsername,
		"repo", cfg.Repository,
	)

	fsSvc := fsService.New()

	oldMetadata, err := fsSvc.LoadJsonMetadata(
		cfg.DataFile,
	)
	if err != nil {
		slog.Error(
			"error loading metadata",
			"error", err,
		)
		os.Exit(1)
	}

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

	packages, err := githubSvc.GetContainers()
	if err != nil {
		slog.Error(
			"error getting packages",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info(
		"Found container packages",
		"count", len(packages),
	)

	slog.Info("Comparing package metadata")

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
		slog.Error(
			"error saving metadata",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info(
		"Wrote packages",
		"count", len(packages),
		"file", cfg.DataFile,
	)
}
