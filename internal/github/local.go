package githubapi

import (
	"fmt"
	"os/exec"
	"strings"
)

// LocalGitContext holds information about the current local repository.
type LocalGitContext struct {
	RepoFullName string // e.g. "owner/repo"
	BranchName   string // e.g. "feature/xyz"
}

// GetLocalGitContext extracts the repository owner/repo and current branch from the local environment.
func GetLocalGitContext() (*LocalGitContext, error) {
	// 1. Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(branchOut))

	// 2. Get remote URL
	remoteCmd := exec.Command("git", "remote", "get-url", "origin")
	remoteOut, err := remoteCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote 'origin': %w (are you in a git repo?)", err)
	}
	remoteURL := strings.TrimSpace(string(remoteOut))

	// 3. Parse owner/repo from URL
	repoFullName, err := parseRepoFromRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}

	return &LocalGitContext{
		RepoFullName: repoFullName,
		BranchName:   branch,
	}, nil
}

// parseRepoFromRemoteURL handles HTTPS and SSH GitHub URLs.
func parseRepoFromRemoteURL(remoteURL string) (string, error) {
	// HTTPS: https://github.com/owner/repo.git
	// SSH: git@github.com:owner/repo.git

	s := strings.TrimSuffix(remoteURL, ".git")

	if strings.HasPrefix(s, "https://github.com/") {
		return strings.TrimPrefix(s, "https://github.com/"), nil
	}

	if strings.Contains(s, "github.com:") {
		parts := strings.SplitN(s, "github.com:", 2)
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("unsupported remote URL format: %s", remoteURL)
}
