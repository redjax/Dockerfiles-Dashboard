package github

type Package struct {
	Name string `json:"name"`
}

type ContainerMetadata struct {
	Tags []string `json:"tags"`
}

type PackageMetadata struct {
	Container ContainerMetadata `json:"container"`
}

type PackageVersion struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Metadata  PackageMetadata `json:"metadata"`
}

type PastVersion struct {
	Tag         string `json:"tag"`
	ReleaseDate string `json:"release_date"`
	Digest      string `json:"digest,omitempty"`
}

type Container struct {
	Image         string        `json:"image"`
	Latest        string        `json:"latest"`
	URL           string        `json:"url"`
	LatestDigest  string        `json:"latest_digest,omitempty"`
	LatestUpdated string        `json:"latest_updated,omitempty"`
	PastVersions  []PastVersion `json:"past_versions"`
}
