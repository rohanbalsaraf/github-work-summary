package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
	"github.com/RDX463/github-work-summary/internal/summary"
)

// Report contains everything needed to render a summary.
type PRIntelligence struct {
	RiskLevel       string   `json:"risk_level"`  // Low, Medium, High
	RiskReason      string   `json:"risk_reason"` // One-line explanation
	RiskAreas       []string `json:"risk_areas"`  // e.g. ["Security", "Database", "Complexity"]
	SuggestedLabels []string `json:"suggested_labels"`
}

// BuildReportPrompt constructs the base instructions and data for any LLM to summarize the work.
func BuildReportPrompt(report summary.Report) string {
	// Try loading custom template first
	home, _ := os.UserHomeDir()
	templatePath := filepath.Join(home, ".gws", "templates", "summary.md")

	if data, err := os.ReadFile(templatePath); err == nil {
		return renderTemplate(string(data), report)
	}

	// Default fallback
	var b strings.Builder
	b.WriteString("You are a professional software engineering manager summarizing a developer's daily work based on their GitHub activity.\n")
	b.WriteString("Analyze the following commit messages and pull request titles and provide a concise, high-impact summary of the day's work.\n")
	b.WriteString("Focus on the 'What' and 'Why' instead of just listing the 'How'. Use professional language.\n\n")

	fmt.Fprintf(&b, "Timeframe: %s to %s\n", report.WindowStart.Format("2006-01-02"), report.WindowEnd.Format("2006-01-02"))
	fmt.Fprintf(&b, "Total Commits: %d\n", report.TotalCommits)
	fmt.Fprintf(&b, "Total Pull Requests: %d\n\n", report.TotalPRs)

	if len(report.TicketInfo) > 0 {
		b.WriteString("Related Tickets (Business Context):\n")
		for _, t := range report.TicketInfo {
			fmt.Fprintf(&b, "- [%s] %s (Status: %s)\n", t.ID, t.Title, t.Status)
		}
		b.WriteString("\n")
	}

	for _, repo := range report.Repositories {
		fmt.Fprintf(&b, "### Repository: %s\n", repo.Repository)

		addCommitsToPrompt(&b, "Features", repo.Features)
		addCommitsToPrompt(&b, "Bug Fixes", repo.BugFixes)
		addCommitsToPrompt(&b, "Maintenance", repo.Maintenance)
		addCommitsToPrompt(&b, "Other", repo.Other)

		if len(repo.PullRequests) > 0 {
			b.WriteString("Pull Requests:\n")
			for _, pr := range repo.PullRequests {
				fmt.Fprintf(&b, "- %s (#%d)\n", pr.Title, pr.Number)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\nSummary requirements:\n")
	b.WriteString("- Keep it under 200 words.\n")
	b.WriteString("- Use 2-3 short paragraphs or bullet points.\n")
	b.WriteString("- Do not use markdown headers (level 1-3) inside your response.\n")
	b.WriteString("- Highlight the key focus of the day (e.g., 'Primary focus was stabilizing the payment integration...').\n")

	return b.String()
}

// BuildTrendPrompt constructs the instructions for an LLM to analyze work trends over a period.
func BuildTrendPrompt(report summary.Report) string {
	var b strings.Builder
	b.WriteString("You are a senior engineering director. Analyze the following work summary data to identify trends, velocity, and focus areas over the given period.\n")
	fmt.Fprintf(&b, "Timeframe: %s to %s\n", report.WindowStart.Format("2006-01-02"), report.WindowEnd.Format("2006-01-02"))
	fmt.Fprintf(&b, "Total Activity: %d Commits, %d PRs\n\n", report.TotalCommits, report.TotalPRs)

	b.WriteString("Analyze the developer's output to identify patterns (feature development vs maintenance vs bug fixes) and bottlenecks. Focus on the 'Strategic Impact' and 'Velocity Trend'.\n\n")

	for _, repo := range report.Repositories {
		fmt.Fprintf(&b, "### Repository: %s\n", repo.Repository)
		addCommitsToPrompt(&b, "Features", repo.Features)
		addCommitsToPrompt(&b, "Fixes", repo.BugFixes)
		addCommitsToPrompt(&b, "Refactor/Maint", repo.Maintenance)
		b.WriteString("\n")
	}

	b.WriteString("\nResponse Requirements:\n")
	b.WriteString("- Highlight the most significant strategic objective achieved.\n")
	b.WriteString("- Identify the 'Primary Focus' profile (e.g. '80% feature development').\n")
	b.WriteString("- Give a concise summary of the velocity trend.\n")
	b.WriteString("- Keep it under 250 words.\n")
	b.WriteString("- Use professional, data-driven language suitable for an engineering leadership review.\n")

	return b.String()
}

// BuildManagerPrompt constructs the instructions for a high-level executive summary.
func BuildManagerPrompt(report summary.Report) string {
	var b strings.Builder
	b.WriteString("You are a VP of Engineering. Summarize the collective work of this team/individual for senior leadership.\n")
	b.WriteString("Focus on 'Business Value', 'Reliability', and 'Strategic Alignment'. Avoid deep technical jargon.\n\n")

	fmt.Fprintf(&b, "Timeframe: %s to %s\n", report.WindowStart.Format("2006-01-02"), report.WindowEnd.Format("2006-01-02"))
	fmt.Fprintf(&b, "Org/Team Activity: %d Commits, %d PRs across %d Repositories.\n\n", report.TotalCommits, report.TotalPRs, len(report.Repositories))

	b.WriteString("Summarize based on these pillars:\n")
	b.WriteString("1. **Executive Highlights**: What were the top 3 wins?\n")
	b.WriteString("2. **System Health**: Did we focus on stability, debt, or features?\n")
	b.WriteString("3. **Strategic Trajectory**: Is the team move closer to its quarterly goals?\n")

	return b.String()
}

// BuildAuditPrompt constructs instructions for an evidence-based performance audit.
func BuildAuditPrompt(report summary.Report) string {
	var b strings.Builder
	b.WriteString("You are a technical auditor. Provide an objective, evidence-based assessment of this developer's contributions for a performance review.\n")
	b.WriteString("Evaluate based on: 'Complexity', 'Quality', and 'Consistency'.\n\n")

	fmt.Fprintf(&b, "Data Source: %d Commits, %d PRs\n\n", report.TotalCommits, report.TotalPRs)

	b.WriteString("Structure the audit as follows:\n")
	b.WriteString("- **Contribution Breadth**: Number of repos and areas touched.\n")
	b.WriteString("- **Technical Depth**: Identify the most complex changes based on commit messages.\n")
	b.WriteString("- **Impact Score**: A concise justification of why this work matters to the organization.\n")

	return b.String()
}

func renderTemplate(tpl string, report summary.Report) string {
	res := tpl
	res = strings.ReplaceAll(res, "{{window_start}}", report.WindowStart.Format("2006-01-02"))
	res = strings.ReplaceAll(res, "{{window_end}}", report.WindowEnd.Format("2006-01-02"))
	res = strings.ReplaceAll(res, "{{total_commits}}", fmt.Sprintf("%d", report.TotalCommits))
	res = strings.ReplaceAll(res, "{{total_prs}}", fmt.Sprintf("%d", report.TotalPRs))

	var activity strings.Builder
	for _, repo := range report.Repositories {
		fmt.Fprintf(&activity, "### %s\n", repo.Repository)
		addCommitsToPrompt(&activity, "Features", repo.Features)
		addCommitsToPrompt(&activity, "Bug Fixes", repo.BugFixes)
		addCommitsToPrompt(&activity, "Maintenance", repo.Maintenance)
		addCommitsToPrompt(&activity, "Other", repo.Other)
		if len(repo.PullRequests) > 0 {
			activity.WriteString("Pull Requests:\n")
			for _, pr := range repo.PullRequests {
				fmt.Fprintf(&activity, "- %s (#%d)\n", pr.Title, pr.Number)
			}
		}
		activity.WriteString("\n")
	}
	res = strings.ReplaceAll(res, "{{activity}}", activity.String())

	var tickets strings.Builder
	for _, t := range report.TicketInfo {
		fmt.Fprintf(&tickets, "- [%s] %s (Status: %s)\n", t.ID, t.Title, t.Status)
	}
	res = strings.ReplaceAll(res, "{{tickets}}", tickets.String())

	return res
}

// BuildPRPrompt constructs the instructions for an LLM to write a professional PR description.
func BuildPRPrompt(branchName string, commits []githubapi.Commit) string {
	var b strings.Builder
	b.WriteString("You are a professional software engineer drafting a high-impact Pull Request (PR) description.\n")
	b.WriteString(fmt.Sprintf("Branch Name: %s\n\n", branchName))
	b.WriteString("Analyze the following commit messages to understand the purpose and impact of the changes:\n")
	for _, c := range commits {
		fmt.Fprintf(&b, "- %s\n", c.Message)
		for _, id := range c.Tickets {
			fmt.Fprintf(&b, "  [Linked Ticket: %s]\n", id)
		}
	}

	b.WriteString("\nGenerate a professional PR description in Markdown format with the following sections:\n")
	b.WriteString("1. **Context/Purpose**: Why are these changes being made?\n")
	b.WriteString("2. **Key Changes**: Boldly highlight the most important technical changes.\n")
	b.WriteString("3. **Side Effects**: Mention any potential risks or areas to watch out for.\n")
	b.WriteString("4. **Testing Status**: A placeholder or AI-suggested testing steps.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Keep the tone professional but high-energy.\n")
	b.WriteString("- Focus on impact and clarify for reviewers.\n")
	b.WriteString("- Do not use level 1 headers (#). Use level 2 (##) or bold text.\n")
	b.WriteString("- Maximum length: 300 words.\n")

	return b.String()
}

// BuildPRTitlePrompt constructs the instructions for an LLM to write a professional PR title.
func BuildPRTitlePrompt(branchName string, commits []githubapi.Commit) string {
	var b strings.Builder
	b.WriteString("Generate a concise, professional Pull Request title (max 60 characters) based on these commits:\n")
	for _, c := range commits {
		fmt.Fprintf(&b, "- %s\n", c.Message)
	}
	b.WriteString("\nFollow the 'feat(scope): description' or 'fix(scope): description' conventional commits format if applicable.\n")
	b.WriteString("Return ONLY the title string, no markdown headers or conversational text.")

	return b.String()
}

// BuildPRIntelligencePrompt constructs a prompt to analyze the risk and labels for a PR.
func BuildPRIntelligencePrompt(commits []githubapi.Commit) string {
	var b strings.Builder
	b.WriteString("Analyze the following commits and provide a structured risk assessment and labeling suggestion for a Pull Request:\n")
	for _, c := range commits {
		fmt.Fprintf(&b, "- %s\n", c.Message)
	}

	b.WriteString("\nReturn ONLY a JSON object with the following fields:\n")
	b.WriteString("- risk_level: \"Low\", \"Medium\", or \"High\"\n")
	b.WriteString("- risk_reason: A single sentence explaining the risk level.\n")
	b.WriteString("- risk_areas: A list of categories involved (e.g., \"Security\", \"UI\", \"API\", \"Database\", \"Testing\", \"Refactor\").\n")
	b.WriteString("- suggested_labels: A list of 1-3 standard GitHub labels (e.g., \"feat\", \"fix\", \"docs\", \"chore\", \"breaking-change\").\n")
	b.WriteString("\nRules:\n")
	b.WriteString("- If modifications involve 'go.mod', 'sql', or 'auth', mark as \"Medium\" or \"High\".\n")
	b.WriteString("- If changes are only docs or tests, mark as \"Low\".\n")
	b.WriteString("- Ensure the JSON is valid and contains no other text.")

	return b.String()
}

func addCommitsToPrompt(b *strings.Builder, category string, commits []githubapi.Commit) {
	if len(commits) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", category)
	for _, c := range commits {
		fmt.Fprintf(b, "- %s\n", c.Message)
	}
}

// getPromptForPersona returns the appropriate prompt based on the requested persona.
func getPromptForPersona(report summary.Report, persona string) string {
	switch strings.ToLower(persona) {
	case "manager", "executive":
		return BuildManagerPrompt(report)
	case "audit", "review":
		return BuildAuditPrompt(report)
	default:
		return BuildReportPrompt(report)
	}
}
