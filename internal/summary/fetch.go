package summary

import (
	"context"
	"fmt"
	"sync"
	"time"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
)

const maxRepoConcurrency = 6

type RepoFetchResult struct {
	Repo    string
	Commits githubapi.BranchCommitResult
	Pulls   []githubapi.PullRequest
	Err     error
}

type RepoBranchStatus struct {
	Scanned []string
	Active  map[string]int
}

func FetchWorkData(ctx context.Context, client githubapi.GitHubClient, selectedRepos []string, author string, since time.Time, branches []string, skipPRs bool) (map[string][]githubapi.Commit, map[string][]githubapi.PullRequest, map[string]RepoBranchStatus, []string, error) {
	repoCommits := make(map[string][]githubapi.Commit)
	repoPulls := make(map[string][]githubapi.PullRequest)
	statusByRepo := make(map[string]RepoBranchStatus)
	results := make(chan RepoFetchResult, len(selectedRepos))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxRepoConcurrency)

	for _, repo := range selectedRepos {
		repoName := repo
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := RepoFetchResult{Repo: repoName}
			commits, err := client.ListCommitsByAuthorSinceByBranches(ctx, repoName, author, since, branches)
			if err != nil {
				res.Err = err
			} else {
				res.Commits = commits
				if !skipPRs {
					pulls, _ := client.ListPullRequestsByAuthorSince(ctx, repoName, author, since)
					res.Pulls = pulls
				}
			}
			results <- res
		}()
	}
	wg.Wait()
	close(results)

	var warnings []string
	for res := range results {
		if res.Err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", res.Repo, res.Err))
		} else {
			repoCommits[res.Repo] = res.Commits.Commits
			repoPulls[res.Repo] = res.Pulls
			active := make(map[string]int)
			for _, c := range res.Commits.Commits {
				for _, b := range c.Branches {
					active[b]++
				}
			}
			statusByRepo[res.Repo] = RepoBranchStatus{Scanned: res.Commits.ScannedBranches, Active: active}
		}
	}
	return repoCommits, repoPulls, statusByRepo, warnings, nil
}

func FetchBranchesAcrossRepos(ctx context.Context, client githubapi.GitHubClient, selectedRepos []string) (map[string]int, []string, error) {
	branchRepoCount := make(map[string]int)
	var wg sync.WaitGroup
	results := make(chan RepoFetchResult, len(selectedRepos))
	sem := make(chan struct{}, maxRepoConcurrency)

	for _, repo := range selectedRepos {
		repoName := repo
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			branches, err := client.ListRepositoryBranches(ctx, repoName)
			res := RepoFetchResult{Repo: repoName}
			if err != nil {
				res.Err = err
			} else {
				res.Commits = githubapi.BranchCommitResult{ScannedBranches: branches}
			}
			results <- res
		}()
	}
	wg.Wait()
	close(results)

	var warnings []string
	for res := range results {
		if res.Err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", res.Repo, res.Err))
		} else {
			for _, b := range res.Commits.ScannedBranches {
				branchRepoCount[b]++
			}
		}
	}
	return branchRepoCount, warnings, nil
}
