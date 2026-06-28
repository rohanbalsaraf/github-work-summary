package summary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
)

// Category represents the type of work performed.
type Category string

const (
	CategoryFeature     Category = "Features"
	CategoryBugFix      Category = "Bug Fixes"
	CategoryMaintenance Category = "Maintenance"
	CategoryOther       Category = "Other"
)

// Report contains everything needed to render a summary.
type Report struct {
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
	TotalCommits int       `json:"total_commits"`
	TotalPRs     int       `json:"total_prs"`

	HealthScore float64  `json:"health_score"` // 0-10
	Bottlenecks []string `json:"bottlenecks"`  // e.g. ["Repo X has high bug-to-feature ratio"]

	Repositories []RepoSummary `json:"repositories"`
	AISummary    string        `json:"ai_summary"`

	Tickets    map[string]string `json:"tickets"`     // Ticket ID -> Title
	TicketInfo []Ticket          `json:"ticket_info"` // Full details
}

// RepoSummary compiles activity for a specific repository.
type RepoSummary struct {
	Repository   string                  `json:"repository"`
	Features     []githubapi.Commit      `json:"features"`
	BugFixes     []githubapi.Commit      `json:"bug_fixes"`
	Maintenance  []githubapi.Commit      `json:"maintenance"`
	Other        []githubapi.Commit      `json:"other"`
	PullRequests []githubapi.PullRequest `json:"pull_requests"`

	HealthScore float64 `json:"health_score"` // 0-10
}

// Ticket represents fetched metadata from Jira or Linear.
type Ticket struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// BuildReport calculates a summary from a slice of commits and PRs.
func BuildReport(commits []githubapi.Commit, pulls []githubapi.PullRequest, start, end time.Time) Report {
	repoSummaries := make(map[string]*RepoSummary)

	for _, commit := range commits {
		repo := commit.RepoName
		if repo == "" {
			repo = "unknown"
		}
		if _, exists := repoSummaries[repo]; !exists {
			repoSummaries[repo] = &RepoSummary{Repository: repo}
		}

		s := repoSummaries[repo]
		msg := strings.ToLower(commit.Message)
		if strings.HasPrefix(msg, "feat") || strings.Contains(msg, "feature") {
			s.Features = append(s.Features, commit)
		} else if strings.HasPrefix(msg, "fix") || strings.Contains(msg, "bug") {
			s.BugFixes = append(s.BugFixes, commit)
		} else if strings.HasPrefix(msg, "chore") || strings.HasPrefix(msg, "refactor") || strings.Contains(msg, "maintenance") {
			s.Maintenance = append(s.Maintenance, commit)
		} else {
			s.Other = append(s.Other, commit)
		}
	}

	for _, pr := range pulls {
		repo := pr.RepoName
		if repo == "" {
			repo = "unknown"
		}
		if _, exists := repoSummaries[repo]; !exists {
			repoSummaries[repo] = &RepoSummary{Repository: repo}
		}
		repoSummaries[repo].PullRequests = append(repoSummaries[repo].PullRequests, pr)
	}

	sortedRepos := make([]RepoSummary, 0, len(repoSummaries))
	for _, s := range repoSummaries {
		sortedRepos = append(sortedRepos, *s)
	}
	sort.Slice(sortedRepos, func(i, j int) bool {
		return sortedRepos[i].Repository < sortedRepos[j].Repository
	})

	// Calculate Health Scores
	var totalHealth float64
	for i := range sortedRepos {
		s := &sortedRepos[i]
		f, b, m, o := len(s.Features), len(s.BugFixes), len(s.Maintenance), len(s.Other)

		// Formula: (Features*2 + Other) / (BugFixes*3 + Maintenance*2 + 1) * 5
		score := float64(f*2+o) / float64(b*3+m*2+1) * 5
		if score > 10 {
			score = 10
		}
		s.HealthScore = score
		totalHealth += score
	}

	avgHealth := 0.0
	if len(sortedRepos) > 0 {
		avgHealth = totalHealth / float64(len(sortedRepos))
	}

	var bottlenecks []string
	for _, s := range sortedRepos {
		if s.HealthScore < 3.0 && (len(s.BugFixes) > 0 || len(s.Maintenance) > 0) {
			bottlenecks = append(bottlenecks, fmt.Sprintf("%s has a high maintenance-to-feature ratio (Health: %.1f)", s.Repository, s.HealthScore))
		}
	}

	return Report{
		WindowStart:  start,
		WindowEnd:    end,
		TotalCommits: len(commits),
		TotalPRs:     len(pulls),
		Repositories: sortedRepos,
		HealthScore:  avgHealth,
		Bottlenecks:  bottlenecks,
		Tickets:      make(map[string]string),
	}
}

// ToMarkdown generates a Markdown version of the report.
func (r *Report) ToMarkdown() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Work Summary (%s - %s)\n\n",
		r.WindowStart.Format("Jan 02, 15:04"),
		r.WindowEnd.Format("Jan 02, 15:04"))

	if r.AISummary != "" {
		b.WriteString("## AI Impact Summary\n")
		b.WriteString(r.AISummary)
		b.WriteString("\n\n")
	}

	if len(r.TicketInfo) > 0 {
		b.WriteString("## Related Tickets\n")
		for _, t := range r.TicketInfo {
			fmt.Fprintf(&b, "- [%s](%s): %s (%s)\n", t.ID, t.URL, t.Title, t.Status)
		}
		b.WriteString("\n")
	}

	for _, repo := range r.Repositories {
		fmt.Fprintf(&b, "### %s\n\n", repo.Repository)

		if len(repo.Features) > 0 {
			b.WriteString("#### Features\n")
			for _, c := range repo.Features {
				fmt.Fprintf(&b, "- %s ([%s](%s))\n", ShortSubject(c.Message), c.SHA[:7], c.HTMLURL)
			}
			b.WriteString("\n")
		}
		if len(repo.BugFixes) > 0 {
			b.WriteString("#### Bug Fixes\n")
			for _, c := range repo.BugFixes {
				fmt.Fprintf(&b, "- %s ([%s](%s))\n", ShortSubject(c.Message), c.SHA[:7], c.HTMLURL)
			}
			b.WriteString("\n")
		}
		if len(repo.PullRequests) > 0 {
			b.WriteString("#### Pull Requests\n")
			for _, p := range repo.PullRequests {
				fmt.Fprintf(&b, "- #%d: %s ([view](%s))\n", p.Number, p.Title, p.HTMLURL)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// ToJSON serializes the report to a JSON byte slice.
func (r *Report) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToHTML generates a styled HTML version of the report.
func (r *Report) ToHTML() (string, error) {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Work Summary</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f9f9f9; }
        .card { background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.05); border: 1px solid #eee; }
        h1 { color: #1a1a1a; border-bottom: 2px solid #eee; padding-bottom: 10px; margin-top: 0; }
        h2 { color: #2c3e50; margin-top: 30px; border-left: 4px solid #3498db; padding-left: 15px; }
        h3 { color: #34495e; border-bottom: 1px solid #eee; padding-bottom: 5px; }
        .ai-box { background: #f0f7ff; border: 1px solid #d0e7ff; padding: 20px; border-radius: 8px; font-style: italic; color: #2c5282; margin: 20px 0; }
        .ticket { background: #fff5f5; border: 1px solid #fed7d7; padding: 10px 15px; border-radius: 6px; margin-bottom: 10px; list-style: none; }
        .commit-list { list-style: none; padding-left: 0; }
        .commit-item { margin-bottom: 8px; display: flex; align-items: baseline; }
        .commit-sha { font-family: monospace; background: #edf2f7; padding: 2px 6px; border-radius: 4px; margin-right: 10px; color: #4a5568; font-size: 0.9em; text-decoration: none; }
        
        /* Trends Chart */
        .trend-chart { display: flex; align-items: flex-end; gap: 4px; height: 100px; margin: 20px 0; border-bottom: 2px solid #eee; padding-bottom: 5px; justify-content: space-around; }
        .trend-bar { background: #3498db; width: 100%; border-radius: 4px 4px 0 0; position: relative; min-width: 10px; max-width: 40px; }
        .trend-bar:hover { background: #2980b9; }
        .trend-label { position: absolute; bottom: -25px; left: 50%; transform: translateX(-50%); font-size: 10px; color: #718096; white-space: nowrap; }
        
        .footer { margin-top: 60px; text-align: center; font-size: 0.8em; color: #a0aec0; }

        /* Health Dashboard */
        .health-dashboard { display: grid; grid-template-columns: 1fr 2fr; gap: 20px; margin: 25px 0; }
        .score-card { background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 8px; padding: 20px; text-align: center; }
        .score-val { font-size: 2.5em; font-weight: bold; color: #2d3748; line-height: 1; }
        .score-label { font-size: 0.9em; color: #718096; margin-top: 5px; }
        .bottleneck-box { background: #fff5f5; border: 1px solid #fed7d7; border-radius: 8px; padding: 15px; }
        .bottleneck-item { color: #c53030; font-size: 0.9em; margin-bottom: 5px; }
        .health-bar-container { width: 100px; height: 8px; background: #edf2f7; border-radius: 4px; overflow: hidden; display: inline-block; vertical-align: middle; margin-left: 10px; }
        .health-bar { height: 100%; }
    </style>
</head>
<body>
    <div class="card">
`)

	fmt.Fprintf(&b, "        <h1>Work Summary</h1>\n")
	fmt.Fprintf(&b, "        <p><strong>Window:</strong> %s &mdash; %s</p>\n",
		r.WindowStart.Format("Jan 02, 2006"),
		r.WindowEnd.Format("Jan 02, 2006"))

	// Health Dashboard
	b.WriteString("        <div class=\"health-dashboard\">\n")
	b.WriteString("            <div class=\"score-card\">\n")
	fmt.Fprintf(&b, "                <div class=\"score-val\">%.1f</div>\n", r.HealthScore)
	b.WriteString("                <div class=\"score-label\">Agility Score (0-10)</div>\n")
	b.WriteString("            </div>\n")

	if len(r.Bottlenecks) > 0 {
		b.WriteString("            <div class=\"bottleneck-box\">\n")
		b.WriteString("                <strong>Predictive Bottlenecks:</strong>\n")
		for _, bt := range r.Bottlenecks {
			fmt.Fprintf(&b, "                <div class=\"bottleneck-item\">⚠ %s</div>\n", bt)
		}
		b.WriteString("            </div>\n")
	} else {
		b.WriteString("            <div class=\"bottleneck-box\" style=\"background: #f0fff4; border-color: #c6f6d5;\">\n")
		b.WriteString("                <strong style=\"color: #276749;\">Path is Clear:</strong>\n")
		b.WriteString("                <div style=\"color: #2f855a; font-size: 0.9em;\">No major development bottlenecks detected. Keep up the momentum!</div>\n")
		b.WriteString("            </div>\n")
	}
	b.WriteString("        </div>\n")

	// Build Trend Chart for multi-day reports
	if r.WindowEnd.Sub(r.WindowStart) > 25*time.Hour {
		r.renderTrendChart(&b)
	}

	if r.AISummary != "" {
		b.WriteString("        <h2>AI Impact Summary</h2>\n")
		fmt.Fprintf(&b, "        <div class=\"ai-box\">%s</div>\n", strings.ReplaceAll(r.AISummary, "\n", "<br>"))
	}

	if len(r.TicketInfo) > 0 {
		b.WriteString("        <h2>Related Tickets</h2>\n")
		b.WriteString("        <ul>\n")
		for _, t := range r.TicketInfo {
			fmt.Fprintf(&b, "            <li class=\"ticket\"><strong>[%s]</strong> <a href=\"%s\">%s</a> (%s)</li>\n", t.ID, t.URL, t.Title, t.Status)
		}
		b.WriteString("        </ul>\n")
	}

	for _, repo := range r.Repositories {
		healthColor := "#48bb78"
		if repo.HealthScore < 7.0 {
			healthColor = "#ecc94b"
		}
		if repo.HealthScore < 4.0 {
			healthColor = "#f56565"
		}

		fmt.Fprintf(&b, "        <h3>%s <div class=\"health-bar-container\"><div class=\"health-bar\" style=\"width: %d%%; background: %s\"></div></div></h3>\n",
			repo.Repository, int(repo.HealthScore*10), healthColor)

		renderHTMLSection(&b, "Features", repo.Features)
		renderHTMLSection(&b, "Bug Fixes", repo.BugFixes)
		renderHTMLSection(&b, "Maintenance", repo.Maintenance)
		renderHTMLSection(&b, "Other", repo.Other)

		if len(repo.PullRequests) > 0 {
			b.WriteString("        <p><strong>Pull Requests:</strong></p>\n")
			b.WriteString("        <ul class=\"commit-list\">\n")
			for _, p := range repo.PullRequests {
				fmt.Fprintf(&b, "            <li class=\"commit-item\">#%d: %s (<a href=\"%s\">view</a>)</li>\n", p.Number, p.Title, p.HTMLURL)
			}
			b.WriteString("        </ul>\n")
		}
	}

	b.WriteString(`
        <div class="footer">
            Generated by github-work-summary | v2.2.3
        </div>
    </div>
</body>
</html>`)

	return b.String(), nil
}

func renderHTMLSection(b *strings.Builder, title string, commits []githubapi.Commit) {
	if len(commits) == 0 {
		return
	}
	fmt.Fprintf(b, "        <p><strong>%s:</strong></p>\n", title)
	b.WriteString("        <ul class=\"commit-list\">\n")
	for _, c := range commits {
		fmt.Fprintf(b, "            <li class=\"commit-item\"><a href=\"%s\" class=\"commit-sha\">%s</a> %s</li>\n", c.HTMLURL, c.SHA[:7], ShortSubject(c.Message))
	}
	b.WriteString("        </ul>\n")
}

func (r *Report) renderTrendChart(b *strings.Builder) {
	b.WriteString("        <div class=\"trend-chart\">\n")

	// Count activity per day
	dayCounts := make(map[string]int)
	var maxCount int

	for _, repo := range r.Repositories {
		allRepoCommits := append([]githubapi.Commit(nil), repo.Features...)
		allRepoCommits = append(allRepoCommits, repo.BugFixes...)
		allRepoCommits = append(allRepoCommits, repo.Maintenance...)
		allRepoCommits = append(allRepoCommits, repo.Other...)

		for _, c := range allRepoCommits {
			d := c.AuthoredAt.Format("01/02")
			dayCounts[d]++
			if dayCounts[d] > maxCount {
				maxCount = dayCounts[d]
			}
		}
	}

	// Sort days
	var days []string
	curr := r.WindowStart
	for curr.Before(r.WindowEnd) || curr.Equal(r.WindowEnd) {
		d := curr.Format("01/02")
		days = append(days, d)
		curr = curr.AddDate(0, 0, 1)
	}

	for _, d := range days {
		count := dayCounts[d]
		height := 0
		if maxCount > 0 {
			height = (count * 100) / maxCount
		}
		if height < 5 && count > 0 {
			height = 5
		} // Min visibility

		fmt.Fprintf(b, "            <div class=\"trend-bar\" style=\"height: %d%%\" title=\"%d activities\">\n", height, count)
		fmt.Fprintf(b, "                <span class=\"trend-label\">%s</span>\n", d)
		b.WriteString("            </div>\n")
	}

	b.WriteString("        </div>\n")
}
