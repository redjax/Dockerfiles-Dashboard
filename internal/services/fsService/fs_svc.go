package fsService

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"local/dockerfiles/internal/github"
)

type Metadata struct {
	LastUpdated string             `json:"last_updated"`
	Repository  string             `json:"repository"`
	URL         string             `json:"url"`
	Packages    []github.Container `json:"packages"`
}

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) LoadJsonMetadata(
	filename string,
) (
	Metadata,
	error,
) {
	data, err := os.ReadFile(
		filename,
	)

	if os.IsNotExist(err) {
		return Metadata{
			Packages: []github.Container{},
		}, nil
	}

	if err != nil {
		return Metadata{}, fmt.Errorf(
			"reading metadata file: %w",
			err,
		)
	}

	var metadata Metadata

	if err := json.Unmarshal(
		data,
		&metadata,
	); err != nil {
		return Metadata{}, fmt.Errorf(
			"parsing metadata file: %w",
			err,
		)
	}

	if metadata.Packages == nil {
		metadata.Packages = []github.Container{}
	}

	return metadata, nil
}

func (s *Service) SaveJsonMetadata(
	filename string,
	repository string,
	repositoryURL string,
	packages []github.Container,
) error {
	metadata := Metadata{
		LastUpdated: time.Now().UTC().Format(
			time.RFC3339,
		),
		Repository: repository,
		URL:        repositoryURL,
		Packages:   packages,
	}

	data, err := json.MarshalIndent(
		metadata,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"encoding metadata: %w",
			err,
		)
	}

	data = append(
		data,
		'\n',
	)

	dir := filepath.Dir(
		filename,
	)

	if err := os.MkdirAll(
		dir,
		0755,
	); err != nil {
		return fmt.Errorf(
			"creating metadata directory: %w",
			err,
		)
	}

	tmp, err := os.CreateTemp(
		dir,
		"containers-*.tmp",
	)
	if err != nil {
		return fmt.Errorf(
			"creating temporary metadata file: %w",
			err,
		)
	}

	tmpName := tmp.Name()

	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf(
			"writing temporary metadata file: %w",
			err,
		)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf(
			"closing temporary metadata file: %w",
			err,
		)
	}

	if err := os.Rename(
		tmpName,
		filename,
	); err != nil {
		return fmt.Errorf(
			"replacing metadata file: %w",
			err,
		)
	}

	return nil
}
