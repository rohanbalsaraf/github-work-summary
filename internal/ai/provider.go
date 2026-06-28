package ai

import (
	"context"
	githubapi "github.com/RDX463/github-work-summary/internal/github"
	"github.com/RDX463/github-work-summary/internal/summary"
)

// Provider defines the interface for AI summarization engines.
type Provider interface {
	// Summarize generates a high-impact summary of the given work report.
	Summarize(ctx context.Context, report summary.Report) (string, error)

	// SummarizeWithPersona generates a summary tailored to a specific audience (manager, audit, etc).
	SummarizeWithPersona(ctx context.Context, report summary.Report, persona string) (string, error)

	// GeneratePRDescription creates a professional pull request description based on the branch and its commits.
	GeneratePRDescription(ctx context.Context, branchName string, commits []githubapi.Commit) (string, error)

	// GeneratePRTitle creates a concise, high-impact title for a pull request.
	GeneratePRTitle(ctx context.Context, branchName string, commits []githubapi.Commit) (string, error)

	// GeneratePRIntelligence performs a risk assessment and suggests labels for a PR.
	GeneratePRIntelligence(ctx context.Context, commits []githubapi.Commit) (PRIntelligence, error)

	// GenerateTrendAnalysis summarizes activity over a longer period.
	GenerateTrendAnalysis(ctx context.Context, report summary.Report) (string, error)

	// GenerateRaw sends a raw prompt to the AI provider.
	GenerateRaw(ctx context.Context, prompt string) (string, error)

	// Name returns the provider's identifier.
	Name() string
}
