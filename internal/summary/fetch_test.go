package summary

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
)

type mockGitHubClient struct {
	githubapi.GitHubClient
	commitsErr  error
	pullsErr    error
	branchesErr error
	commits     githubapi.BranchCommitResult
	pulls       []githubapi.PullRequest
	branches    []string
}

func (m *mockGitHubClient) ListCommitsByAuthorSinceByBranches(ctx context.Context, repo, author string, since time.Time, branches []string) (githubapi.BranchCommitResult, error) {
	if m.commitsErr != nil {
		return githubapi.BranchCommitResult{}, m.commitsErr
	}
	return m.commits, nil
}

func (m *mockGitHubClient) ListPullRequestsByAuthorSince(ctx context.Context, repo, author string, since time.Time) ([]githubapi.PullRequest, error) {
	if m.pullsErr != nil {
		return nil, m.pullsErr
	}
	return m.pulls, nil
}

func (m *mockGitHubClient) ListRepositoryBranches(ctx context.Context, repo string) ([]string, error) {
	if m.branchesErr != nil {
		return nil, m.branchesErr
	}
	return m.branches, nil
}

func TestFetchWorkData(t *testing.T) {
	mockClient := &mockGitHubClient{
		commits: githubapi.BranchCommitResult{
			Commits: []githubapi.Commit{
				{SHA: "123", Message: "test commit", Branches: []string{"main"}},
			},
			ScannedBranches: []string{"main"},
		},
		pulls: []githubapi.PullRequest{
			{Title: "test PR"},
		},
	}

	ctx := context.Background()
	repos := []string{"owner/repo1"}
	author := "testuser"
	since := time.Now().Add(-24 * time.Hour)
	branches := []string{"main"}

	repoCommits, repoPulls, statusByRepo, warnings, err := FetchWorkData(ctx, mockClient, repos, author, since, branches, false)

	if err != nil {
		t.Fatalf("FetchWorkData failed: %v", err)
	}

	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	if len(repoCommits) != 1 || len(repoCommits["owner/repo1"]) != 1 {
		t.Errorf("expected 1 commit for owner/repo1, got %v", repoCommits)
	}

	if len(repoPulls) != 1 || len(repoPulls["owner/repo1"]) != 1 {
		t.Errorf("expected 1 PR for owner/repo1, got %v", repoPulls)
	}

	status, ok := statusByRepo["owner/repo1"]
	if !ok {
		t.Fatal("expected status for owner/repo1")
	}
	if !reflect.DeepEqual(status.Scanned, []string{"main"}) {
		t.Errorf("expected Scanned branches [main], got %v", status.Scanned)
	}
	if status.Active["main"] != 1 {
		t.Errorf("expected 1 active commit on main, got %v", status.Active)
	}
}

func TestFetchWorkData_Error(t *testing.T) {
	mockClient := &mockGitHubClient{
		commitsErr: errors.New("API limit exceeded"),
	}

	ctx := context.Background()
	repos := []string{"owner/repo1"}
	author := "testuser"
	since := time.Now().Add(-24 * time.Hour)
	branches := []string{"main"}

	repoCommits, repoPulls, _, warnings, err := FetchWorkData(ctx, mockClient, repos, author, since, branches, false)

	if err != nil {
		t.Fatalf("FetchWorkData should not return global error on repo failure, got: %v", err)
	}

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}

	if len(repoCommits) != 0 {
		t.Errorf("expected no commits, got %v", repoCommits)
	}
	if len(repoPulls) != 0 {
		t.Errorf("expected no pulls, got %v", repoPulls)
	}
}

func TestFetchBranchesAcrossRepos(t *testing.T) {
	mockClient := &mockGitHubClient{
		branches: []string{"main", "develop"},
	}

	ctx := context.Background()
	repos := []string{"owner/repo1", "owner/repo2"}

	branchRepoCount, warnings, err := FetchBranchesAcrossRepos(ctx, mockClient, repos)
	if err != nil {
		t.Fatalf("FetchBranchesAcrossRepos failed: %v", err)
	}

	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	if branchRepoCount["main"] != 2 {
		t.Errorf("expected main branch count 2, got %v", branchRepoCount["main"])
	}
	if branchRepoCount["develop"] != 2 {
		t.Errorf("expected develop branch count 2, got %v", branchRepoCount["develop"])
	}
}
