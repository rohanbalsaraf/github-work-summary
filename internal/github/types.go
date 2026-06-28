package githubapi

import (
	"context"
	"time"
)

// User is the authenticated GitHub user.
type User struct {
	Login string `json:"login"`
}

// Repository is a minimal GitHub repository view needed by this CLI.
type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	Fork     bool   `json:"fork"`
	Archived bool   `json:"archived"`
	HTMLURL  string `json:"html_url"`
}

// Commit is a minimal commit payload needed by the summary command.
type Commit struct {
	SHA        string
	Message    string
	HTMLURL    string
	AuthoredAt time.Time
	Branches   []string
	RepoName   string   // e.g. "owner/repo"
	Tickets    []string // List of ticket IDs found in this commit
}

// GitHubClient is an interface representing the GitHub API operations needed by the tool.
type GitHubClient interface {
	ListAccessibleRepositories(ctx context.Context) ([]Repository, error)
	ListOrgRepositories(ctx context.Context, org string) ([]Repository, error)
	GetAuthenticatedUser(ctx context.Context) (User, error)
	ListCommitsByAuthorSinceByBranches(ctx context.Context, repo, author string, since time.Time, branches []string) (BranchCommitResult, error)
	ListRepositoryBranches(ctx context.Context, repo string) ([]string, error)
	ListPullRequestsByAuthorSince(ctx context.Context, repo, author string, since time.Time) ([]PullRequest, error)

	// CreatePullRequest creates a new draft or public PR on GitHub.
	CreatePullRequest(ctx context.Context, repo, head, base, title, body string, draft bool) (int, string, error)

	// AddLabelsToIssue applies a set of labels to a GitHub issue (or PR).
	AddLabelsToIssue(ctx context.Context, repo string, number int, labels []string) error

	// GetCompareCommits fetches the list of commits unique to the head branch relative to the base branch.
	GetCompareCommits(ctx context.Context, repo, base, head string) ([]Commit, error)
}

// PullRequest represents a GitHub PR.
type PullRequest struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Locked    bool       `json:"locked"`
	User      User       `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	MergedAt  *time.Time `json:"merged_at"`
	HTMLURL   string     `json:"html_url"`
	RepoName  string     // e.g. "owner/repo"
}
