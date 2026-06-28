package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/RDX463/github-work-summary/internal/summary"
)

// AgenticSummarize implements a multi-agent map-reduce summarization flow.
// It summarizes each repository concurrently and then synthesizes a master report.
func AgenticSummarize(ctx context.Context, provider Provider, report summary.Report, persona string, isTrend bool) (string, error) {
	if len(report.Repositories) == 0 {
		return "", nil
	}

	// If there's only 1 repository, fall back to standard one-shot summarization
	// to save tokens and time.
	if len(report.Repositories) == 1 {
		if isTrend {
			return provider.GenerateTrendAnalysis(ctx, report)
		}
		return provider.SummarizeWithPersona(ctx, report, persona)
	}

	// 1. Map Phase: Summarize each repository concurrently.
	var wg sync.WaitGroup
	type repoResult struct {
		Repo    string
		Summary string
		Err     error
	}

	results := make(chan repoResult, len(report.Repositories))

	for _, repo := range report.Repositories {
		wg.Add(1)
		go func(r summary.RepoSummary) {
			defer wg.Done()

			// Build a sub-report for just this repo to prompt the AI.
			subReport := summary.Report{
				WindowStart:  report.WindowStart,
				WindowEnd:    report.WindowEnd,
				TotalCommits: len(r.Features) + len(r.BugFixes) + len(r.Maintenance) + len(r.Other),
				TotalPRs:     len(r.PullRequests),
				Repositories: []summary.RepoSummary{r},
				TicketInfo:   report.TicketInfo,
			}

			prompt := BuildReportPrompt(subReport)
			// Instruct the AI to focus only on this repository
			prompt += "\n\nCRITICAL INSTRUCTION: Focus your summary ONLY on the " + r.Repository + " repository. Keep it concise."

			raw, err := provider.GenerateRaw(ctx, prompt)
			results <- repoResult{Repo: r.Repository, Summary: raw, Err: err}
		}(repo)
	}

	wg.Wait()
	close(results)

	// 2. Reduce Phase: Synthesize the final master report
	var synthesisParts []string
	var errs []string

	for res := range results {
		if res.Err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", res.Repo, res.Err))
		} else {
			synthesisParts = append(synthesisParts, fmt.Sprintf("Repository: %s\n%s\n", res.Repo, res.Summary))
		}
	}

	if len(synthesisParts) == 0 {
		return "", fmt.Errorf("all repository summarizations failed: %s", strings.Join(errs, ", "))
	}

	// Build the synthesis prompt
	var instructions string
	if isTrend {
		instructions = "Focus on long-term trends, velocity changes, and major architectural shifts across the repositories."
	} else if persona == "audit" {
		instructions = "Focus strictly on compliance, security, bug fixes, and quantifiable metrics across the repositories. Omit fluff."
	} else if persona != "default" && persona != "" {
		instructions = fmt.Sprintf("Write this synthesis specifically for a %s audience.", persona)
	} else {
		instructions = "Find the overarching themes, major cross-repository features, and highlight the most critical engineering work."
	}

	synthesisPrompt := fmt.Sprintf(`You are an expert technical program manager synthesizing a multi-repository engineering report.
Below are the individual AI-generated summaries for each repository worked on during this period.

Synthesize these into a single, cohesive, high-impact executive summary. 
Do not just concatenate them. %s
Keep it concise, professional, and formatted in Markdown.

--- REPOSITORY SUMMARIES ---
%s
`, instructions, strings.Join(synthesisParts, "\n---\n"))

	masterSummary, err := provider.GenerateRaw(ctx, synthesisPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to synthesize master report: %w", err)
	}

	// If there were any errors in the map phase, append them as warnings.
	if len(errs) > 0 {
		masterSummary += fmt.Sprintf("\n\n*(Note: Some repositories failed to summarize: %s)*", strings.Join(errs, ", "))
	}

	return masterSummary, nil
}
