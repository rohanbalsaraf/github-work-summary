package summary

import (
	"bytes"
	"strings"
	"testing"
	"time"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
)

func TestRender(t *testing.T) {
	windowStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	report := Report{
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		TotalCommits: 2,
		TotalPRs:     1,
		AISummary:    "This is an AI summary of your excellent work.",
		TicketInfo: []Ticket{
			{ID: "PROJ-123", Title: "Fix login bug", Status: "Done"},
		},
		Repositories: []RepoSummary{
			{
				Repository: "owner/repo",
				Features: []githubapi.Commit{
					{SHA: "abcdef123", Message: "feat: add login", Branches: []string{"main"}, AuthoredAt: windowStart.Add(1 * time.Hour)},
				},
				PullRequests: []githubapi.PullRequest{
					{Number: 1, Title: "Add login", State: "open", HTMLURL: "https://github.com"},
				},
			},
		},
	}

	var buf bytes.Buffer
	Render(&buf, report)

	output := buf.String()

	// Assertions to ensure core elements are rendered
	if !strings.Contains(output, "GitHub Work Summary") {
		t.Errorf("Render missing title")
	}
	if !strings.Contains(output, "2 Commits, 1 PRs") {
		t.Errorf("Render missing total activity count")
	}
	if !strings.Contains(output, "AI IMPACT SUMMARY") {
		t.Errorf("Render missing AI header")
	}
	if !strings.Contains(output, "This is an AI summary of your excellent work.") {
		t.Errorf("Render missing AI content")
	}
	if !strings.Contains(output, "PROJ-123") || !strings.Contains(output, "Fix login bug") {
		t.Errorf("Render missing Ticket info")
	}
	if !strings.Contains(output, "feat: add login") {
		t.Errorf("Render missing commit message")
	}
	if !strings.Contains(output, "Add login") {
		t.Errorf("Render missing PR title")
	}
}

func TestRender_EmptyReport(t *testing.T) {
	windowStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	report := Report{
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		TotalCommits: 0,
		TotalPRs:     0,
		Repositories: []RepoSummary{
			{Repository: "owner/repo"},
		},
	}

	var buf bytes.Buffer
	Render(&buf, report)
	output := buf.String()

	if !strings.Contains(output, "No commits in the last") {
		t.Errorf("Render missing empty commits message, got: %s", output)
	}
}

func TestRender_NoRepos(t *testing.T) {
	report := Report{}

	var buf bytes.Buffer
	Render(&buf, report)
	output := buf.String()

	if !strings.Contains(output, "No repositories selected") {
		t.Errorf("Render missing no repos message, got: %s", output)
	}
}
