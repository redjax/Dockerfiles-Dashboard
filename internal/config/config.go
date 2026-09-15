package config

import (
	"flag"
	"fmt"
	"io"
	"local/dockerfiles/internal/logging"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDataFile = "dockerfiles.metadata.json"
	defaultWorkers  = 4
	defaultUsername = "redjax"
	defaultRepo     = "Dockerfiles"
)

type Config struct {
	GithubUsername string
	Repository     string
	GithubToken    string
	TokenFile      string
	DataFile       string
	LogFile        string
	LogLevel       string
	Workers        int
}

func LoadConfig() (Config, error) {
	var cfg Config

	flag.StringVar(
		&cfg.GithubUsername,
		"github-user",
		defaultUsername,
		"Github username",
	)

	flag.StringVar(
		&cfg.Repository,
		"repo",
		defaultRepo,
		"Github repository",
	)

	flag.StringVar(
		&cfg.GithubToken,
		"token",
		"",
		"Github token, or '-' to read the token from stdin",
	)

	flag.StringVar(
		&cfg.TokenFile,
		"token-file",
		"",
		"Read the Github token from a file",
	)

	flag.StringVar(
		&cfg.DataFile,
		"data-file",
		defaultDataFile,
		"Path to the JSON metadata file",
	)

	flag.StringVar(
		&cfg.LogFile,
		"log-file",
		"",
		"Write logs to this file in addition to stdout",
	)

	flag.StringVar(
		&cfg.LogLevel,
		"log-level",
		os.Getenv("LOG_LEVEL"),
		"Log level: debug, info, warn, or error",
	)

	flag.IntVar(
		&cfg.Workers,
		"workers",
		defaultWorkers,
		"Number of concurrent Github requests",
	)

	flag.Parse()

	if cfg.Workers < 1 {
		return Config{}, fmt.Errorf(
			"--workers must be at least 1",
		)
	}

	if strings.TrimSpace(cfg.GithubUsername) == "" {
		return Config{}, fmt.Errorf(
			"--github-user cannot be empty",
		)
	}

	if strings.TrimSpace(cfg.Repository) == "" {
		return Config{}, fmt.Errorf(
			"--repo cannot be empty",
		)
	}

	if _, err := logging.ParseLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}

	token, err := resolveToken(
		cfg.GithubToken,
		cfg.TokenFile,
	)
	if err != nil {
		return Config{}, err
	}

	cfg.GithubToken = token

	return cfg, nil
}

func resolveToken(
	tokenArg string,
	tokenFile string,
) (string, error) {
	switch {
	case tokenFile != "":
		tokenFile, err := expandPath(tokenFile)
		if err != nil {
			return "", fmt.Errorf(
				"expanding token file path: %w",
				err,
			)
		}

		data, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", fmt.Errorf(
				"reading token file: %w",
				err,
			)
		}

		return strings.TrimSpace(string(data)), nil

	case tokenArg == "-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf(
				"reading token from stdin: %w",
				err,
			)
		}

		return strings.TrimSpace(string(data)), nil

	case tokenArg != "":
		return strings.TrimSpace(tokenArg), nil

	default:
		// An empty token is valid. Github will apply the unauthenticated API rate limit.
		return strings.TrimSpace(
			os.Getenv("GITHUB_TOKEN"),
		), nil
	}
}

func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		if path == "~" {
			return home, nil
		}

		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}
