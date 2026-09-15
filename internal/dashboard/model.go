package dashboard

import (
	"time"

	"local/dockerfiles/internal/services/fsService"
)

type PageData struct {
	Metadata       fsService.Metadata
	ContainerCount int
	UpdatedCount   int
	LastUpdated    time.Time
}

func BuildPageData(
	metadata fsService.Metadata,
) PageData {
	var lastUpdated time.Time

	if metadata.LastUpdated != "" {
		lastUpdated, _ = time.Parse(
			time.RFC3339,
			metadata.LastUpdated,
		)
	}

	updatedCount := 0

	for _, container := range metadata.Packages {
		if container.LatestUpdated != "" {
			updatedCount++
		}
	}

	return PageData{
		Metadata:       metadata,
		ContainerCount: len(metadata.Packages),
		UpdatedCount:   updatedCount,
		LastUpdated:    lastUpdated,
	}
}
