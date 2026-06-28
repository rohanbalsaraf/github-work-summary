package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// type PullRequest is now in types.go

// ListPullRequestsByAuthorSince fetches PRs for a repository that were updated after since.
func (c *Client) ListPullRequestsByAuthorSince(ctx context.Context, repo, author string, since time.Time) ([]PullRequest, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/repos/%s/pulls", c.baseURL, repo))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pulls endpoint: %w", err)
	}

	q := endpoint.Query()
	q.Set("state", "all")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	q.Set("per_page", "100")
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pulls request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pulls request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseBodyBytes))
		return nil, parseAPIError(resp.StatusCode, body)
	}

	var all []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("failed to parse pulls response: %w", err)
	}

	var filtered []PullRequest
	for _, pr := range all {
		// Filter by author and timestamp
		// We use UpdatedAt because a PR might have been merged/closed in our window even if created earlier.
		if pr.User.Login == author && (pr.UpdatedAt.After(since) || pr.UpdatedAt.Equal(since)) {
			pr.RepoName = repo
			filtered = append(filtered, pr)
		}
		// Since we sorted by updated desc, we can stop if we reach older PRs.
		if pr.UpdatedAt.Before(since) {
			break
		}
	}

	return filtered, nil
}

// CreatePullRequest submits a new PR to GitHub.
func (c *Client) CreatePullRequest(ctx context.Context, repo, head, base, title, body string, draft bool) (int, string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/repos/%s/pulls", c.baseURL, repo))
	if err != nil {
		return 0, "", err
	}

	payload := map[string]interface{}{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
		"draft": draft,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseBodyBytes))
		return 0, "", parseAPIError(resp.StatusCode, respBody)
	}

	var result struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", err
	}

	return result.Number, result.HTMLURL, nil
}

// AddLabelsToIssue applies a set of labels to a GitHub issue (or PR).
func (c *Client) AddLabelsToIssue(ctx context.Context, repo string, number int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}

	u, err := url.Parse(fmt.Sprintf("%s/repos/%s/issues/%d/labels", c.baseURL, repo, number))
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"labels": labels,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseBodyBytes))
		return parseAPIError(resp.StatusCode, respBody)
	}

	return nil
}
