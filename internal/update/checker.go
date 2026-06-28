package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	repoOwner                = "RDX463"
	repoName                 = "github-work-summary"
	latestReleaseURLTemplate = "https://api.github.com/repos/%s/%s/releases/latest"
)

// Info represents update availability.
type Info struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	URL             string
	Changes         []string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// Check fetches the latest version from GitHub and compares it with the local version.
func Check(ctx context.Context, repo, current string) (*Info, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Default to project defaults if not provided
	owner := repoOwner
	name := repoName
	if repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			owner = parts[0]
			name = parts[1]
		}
	}

	url := fmt.Sprintf(latestReleaseURLTemplate, owner, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch latest release: %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	latest := release.TagName

	// Ensure tags have 'v' prefix for semver package
	sCurrent := current
	if !strings.HasPrefix(sCurrent, "v") {
		sCurrent = "v" + sCurrent
	}
	sLatest := latest
	if !strings.HasPrefix(sLatest, "v") {
		sLatest = "v" + sLatest
	}

	available := semver.Compare(sLatest, sCurrent) > 0

	// Parse some changes from the release body (first 3 lines)
	var changes []string
	lines := strings.Split(release.Body, "\n")
	for i := 0; i < len(lines) && len(changes) < 3; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" && (strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*")) {
			changes = append(changes, strings.TrimSpace(line[1:]))
		}
	}

	return &Info{
		CurrentVersion:  current,
		LatestVersion:   release.TagName,
		UpdateAvailable: available,
		URL:             release.HTMLURL,
		Changes:         changes,
	}, nil
}
