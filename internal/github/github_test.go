package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListAccessibleRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			t.Errorf("expected path /user/repos, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected auth header Bearer test-token, got %s", r.Header.Get("Authorization"))
		}

		repos := []Repository{
			{FullName: "owner/repo1", Private: false},
			{FullName: "owner/repo2", Private: true},
		}
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	repos, err := client.ListAccessibleRepositories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Errorf("expected owner/repo1, got %s", repos[0].FullName)
	}
}

func TestGetAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("expected path /user, got %s", r.URL.Path)
		}
		user := User{Login: "testuser"}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	user, err := client.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("expected testuser, got %s", user.Login)
	}
}

func TestListRepositoryBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/branches" {
			t.Errorf("expected path /repos/owner/repo/branches, got %s", r.URL.Path)
		}
		branches := []repoBranchListItem{
			{Name: "main"},
			{Name: "develop"},
		}
		_ = json.NewEncoder(w).Encode(branches)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	branches, err := client.ListRepositoryBranches(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(branches))
	}
	if branches[0] != "develop" { // sorted alphabetically in implementation
		t.Errorf("expected develop, got %s", branches[0])
	}
}

func TestListCommitsByAuthorSinceByBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock branches call first
		if r.URL.Path == "/repos/owner/repo/branches" {
			_ = json.NewEncoder(w).Encode([]repoBranchListItem{{Name: "main"}})
			return
		}
		// Mock commits call
		if r.URL.Path == "/repos/owner/repo/commits" {
			commits := []userCommitListItem{
				{
					SHA:     "sha1",
					HTMLURL: "url1",
					Commit: struct {
						Message string `json:"message"`
						Author  struct {
							Date string `json:"date"`
						} `json:"author"`
					}{
						Message: "feat: commit 1",
						Author: struct {
							Date string `json:"date"`
						}{Date: time.Now().Format(time.RFC3339)},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(commits)
			return
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	result, err := client.ListCommitsByAuthorSinceByBranches(context.Background(), "owner/repo", "testuser", time.Now().Add(-24*time.Hour), []string{"main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(result.Commits))
	}
	if result.Commits[0].Message != "feat: commit 1" {
		t.Errorf("expected feat: commit 1, got %s", result.Commits[0].Message)
	}
}
