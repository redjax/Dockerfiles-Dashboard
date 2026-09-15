package githubService

import (
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"local/dockerfiles/internal/github"
)

type Service struct {
	client   *github.Client
	username string
	repo     string
	workers  int
}

func New(
	client *github.Client,
	username string,
	repo string,
	workers int,
) *Service {
	if workers < 1 {
		workers = 1
	}

	return &Service{
		client:   client,
		username: username,
		repo:     repo,
		workers:  workers,
	}
}

func (s *Service) GetContainers() (
	[]github.Container,
	error,
) {
	packages, err := s.getPackages()
	if err != nil {
		return nil, err
	}

	return s.getContainers(
		packages,
	)
}

func (s *Service) getPackages() (
	[]github.Package,
	error,
) {
	packages := []github.Package{}

	for page := 1; ; page++ {
		path := fmt.Sprintf(
			"/user/packages?package_type=container&per_page=100&page=%d",
			page,
		)

		var pagePackages []github.Package

		if err := s.client.Get(
			path,
			&pagePackages,
		); err != nil {
			return nil, err
		}

		if len(pagePackages) == 0 {
			break
		}

		packages = append(
			packages,
			pagePackages...,
		)
	}

	return packages, nil
}

type containerResult struct {
	pkg       github.Package
	container github.Container
	err       error
}

func (s *Service) getContainers(
	packages []github.Package,
) (
	[]github.Container,
	error,
) {
	if len(packages) == 0 {
		return []github.Container{}, nil
	}

	workerCount := s.workers
	if workerCount > len(packages) {
		workerCount = len(packages)
	}

	jobs := make(
		chan github.Package,
	)

	results := make(
		chan containerResult,
	)

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for pkg := range jobs {
				slog.Info(
					"checking package",
					"package",
					pkg.Name,
				)

				versions, err := s.getPackageVersions(
					pkg.Name,
				)
				if err != nil {
					results <- containerResult{
						pkg: pkg,
						err: err,
					}

					continue
				}

				results <- containerResult{
					pkg: pkg,
					container: buildContainer(
						pkg,
						versions,
						s.username,
						s.repo,
					),
				}
			}
		}()
	}

	go func() {
		for _, pkg := range packages {
			jobs <- pkg
		}

		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	containers := make(
		[]github.Container,
		0,
		len(packages),
	)

	errors := []string{}

	for result := range results {
		if result.err != nil {
			slog.Error(
				"failed to retrieve package",
				"package",
				result.pkg.Name,
				"error",
				result.err,
			)

			errors = append(
				errors,
				fmt.Sprintf(
					"%s: %v",
					result.pkg.Name,
					result.err,
				),
			)

			continue
		}

		containers = append(
			containers,
			result.container,
		)
	}

	sort.Slice(
		containers,
		func(i, j int) bool {
			return strings.ToLower(
				containers[i].Image,
			) < strings.ToLower(
				containers[j].Image,
			)
		},
	)

	if len(errors) > 0 {
		return containers, fmt.Errorf(
			"failed to retrieve %d package(s): %s",
			len(errors),
			strings.Join(errors, "; "),
		)
	}

	return containers, nil
}

func (s *Service) getPackageVersions(
	packageName string,
) (
	[]github.PackageVersion,
	error,
) {
	versions := []github.PackageVersion{}

	encodedName := url.PathEscape(
		packageName,
	)

	for page := 1; ; page++ {
		path := fmt.Sprintf(
			"/user/packages/container/%s/versions?state=active&per_page=100&page=%d",
			encodedName,
			page,
		)

		var pageVersions []github.PackageVersion

		if err := s.client.Get(
			path,
			&pageVersions,
		); err != nil {
			return nil, err
		}

		if len(pageVersions) == 0 {
			break
		}

		versions = append(
			versions,
			pageVersions...,
		)
	}

	return versions, nil
}

func buildContainer(
	pkg github.Package,
	versions []github.PackageVersion,
	username string,
	repository string,
) github.Container {
	sort.Slice(
		versions,
		func(i, j int) bool {
			return versions[i].CreatedAt >
				versions[j].CreatedAt
		},
	)

	container := github.Container{
		Image: pkg.Name,
		URL: fmt.Sprintf(
			"https://github.com/%s/%s/pkgs/container/%s",
			username,
			repository,
			url.PathEscape(pkg.Name),
		),
		PastVersions: []github.PastVersion{},
	}

	latestVersion := findLatestVersion(
		versions,
	)

	if latestVersion != nil {
		container.Latest = selectReleaseTag(
			latestVersion.Metadata.Container.Tags,
		)

		container.LatestDigest = latestVersion.Name

		container.LatestUpdated = formatDate(
			latestVersion.CreatedAt,
		)
	}

	seen := make(
		map[string]struct{},
	)

	for _, version := range versions {
		for _, tag := range version.Metadata.Container.Tags {
			if tag == "latest" {
				continue
			}

			if _, exists := seen[tag]; exists {
				continue
			}

			seen[tag] = struct{}{}

			container.PastVersions = append(
				container.PastVersions,
				github.PastVersion{
					Tag: tag,
					ReleaseDate: formatDate(
						version.CreatedAt,
					),
					Digest: version.Name,
				},
			)
		}
	}

	return container
}

func findLatestVersion(
	versions []github.PackageVersion,
) *github.PackageVersion {
	for i := range versions {
		version := &versions[i]

		for _, tag := range version.Metadata.Container.Tags {
			if tag == "latest" {
				return version
			}
		}
	}

	return nil
}

func selectReleaseTag(
	tags []string,
) string {
	for _, tag := range tags {
		if tag != "latest" {
			return tag
		}
	}

	return "latest"
}

func formatDate(
	value string,
) string {
	if value == "" {
		return ""
	}

	t, err := time.Parse(
		time.RFC3339,
		value,
	)
	if err != nil {
		return value
	}

	return t.UTC().Format(
		time.RFC3339,
	)
}

func Compare(
	old []github.Container,
	current []github.Container,
) {
	oldByImage := make(
		map[string]github.Container,
		len(old),
	)

	currentByImage := make(
		map[string]github.Container,
		len(current),
	)

	for _, container := range old {
		oldByImage[container.Image] = container
	}

	for _, container := range current {
		currentByImage[container.Image] = container

		previous, exists := oldByImage[container.Image]
		if !exists {
			slog.Info(
				"new package",
				"image",
				container.Image,
				"latest",
				container.Latest,
			)

			continue
		}

		if previous.Latest != container.Latest {
			slog.Info(
				"package updated",
				"image",
				container.Image,
				"previous",
				previous.Latest,
				"current",
				container.Latest,
			)
		}

		if previous.LatestDigest != container.LatestDigest {
			slog.Info(
				"package digest changed",
				"image",
				container.Image,
				"previous",
				previous.LatestDigest,
				"current",
				container.LatestDigest,
			)
		}
	}

	for _, container := range old {
		if _, exists := currentByImage[container.Image]; exists {
			continue
		}

		slog.Info(
			"package removed",
			"image",
			container.Image,
		)
	}
}
