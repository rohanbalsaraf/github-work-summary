package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/RDX463/github-work-summary/internal/ai"
	"github.com/RDX463/github-work-summary/internal/auth"
	githubapi "github.com/RDX463/github-work-summary/internal/github"
	"github.com/RDX463/github-work-summary/internal/summary"
	"github.com/RDX463/github-work-summary/internal/tickets"
	"github.com/RDX463/github-work-summary/internal/tui"
	"github.com/RDX463/github-work-summary/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path/filepath"
)

const (
	defaultSummaryWindow  = 24 * time.Hour
	fallbackSummaryWindow = 30 * 24 * time.Hour
	maxRepoConcurrency    = 6
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Fetch and summarize your commits from the last 24 hours",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSummary(cmd)
	},
}

var summaryBranches []string
var summaryChooseBranch bool
var summarySince string
var summaryUntil string
var summaryDuration string
var summaryPickRepos bool
var summaryOutputFile string
var summaryMarkdown bool
var summaryJSON bool
var summaryHTML bool
var summarySkipPRs bool
var summaryAI bool
var summaryShare string
var summaryInteractive bool
var summaryDays int
var summaryOrg string
var summaryPersona string
var summaryAudit bool

func init() {
	rootCmd.AddCommand(summaryCmd)
	summaryCmd.Flags().StringSliceVarP(&summaryBranches, "branch", "b", nil, "Branch name(s) to include")
	summaryCmd.Flags().BoolVar(&summaryChooseBranch, "choose-branch", false, "Open interactive branch selector")

	summaryCmd.Flags().StringVarP(&summarySince, "since", "s", "", "Start date or relative duration (e.g., '2024-03-20' or '2d')")
	summaryCmd.Flags().StringVarP(&summaryUntil, "until", "u", "", "End date (e.g., '2024-03-21')")
	summaryCmd.Flags().StringVarP(&summaryDuration, "duration", "d", "", "Time window size (e.g., '48h', '3d', '1w')")
	summaryCmd.Flags().BoolVar(&summaryPickRepos, "pick", false, "Manually select repositories instead of using saved ones")

	summaryCmd.Flags().StringVarP(&summaryOutputFile, "output", "o", "", "File to write the summary to")
	summaryCmd.Flags().BoolVarP(&summaryMarkdown, "markdown", "m", false, "Output in Markdown format")
	summaryCmd.Flags().BoolVar(&summaryJSON, "json", false, "Output in JSON format")
	summaryCmd.Flags().BoolVar(&summaryHTML, "html", false, "Output in HTML format")
	summaryCmd.Flags().BoolVar(&summarySkipPRs, "no-prs", false, "Exclude Pull Requests from the summary")
	summaryCmd.Flags().BoolVar(&summaryAI, "ai", false, "Generate a professional AI impact summary")
	summaryCmd.Flags().StringVar(&summaryShare, "share", "", "Share the summary directly to Slack or Discord (e.g. --share slack)")
	summaryCmd.Flags().BoolVarP(&summaryInteractive, "interactive", "i", false, "Open interactive dashboard to review and edit summary")
	summaryCmd.Flags().IntVarP(&summaryDays, "days", "n", 0, "Number of days to summarize (e.g. 7, 30)")
	summaryCmd.Flags().StringVar(&summaryOrg, "org", "", "Summarize work for an entire GitHub Organization")
	summaryCmd.Flags().StringVar(&summaryPersona, "persona", "default", "AI persona for the summary (default, manager, audit)")
	summaryCmd.Flags().BoolVar(&summaryAudit, "audit", false, "Enable performance audit mode (shorthand for --persona audit)")
}

func runSummary(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	profileName := getActiveProfileName()
	fmt.Fprintf(out, "%s %s\n", ui.Bold(out, "Profile:"), ui.Cyan(out, profileName))

	client, err := loadGitHubClientFromKeychain()
	if err != nil {
		return err
	}

	repos, err := client.ListAccessibleRepositories(cmd.Context())
	if err != nil {
		if errors.Is(err, githubapi.ErrUnauthorized) {
			return fmt.Errorf("stored token is invalid or expired. run `github-work-summary login` again")
		}
		return err
	}
	if len(repos) == 0 {
		fmt.Fprintln(out, ui.Yellow(out, "No repositories found for this account."))
		return nil
	}

	selectedRepos := viper.GetStringSlice(getProfileKey(profileName, "repositories"))

	if summaryOrg != "" {
		fmt.Fprintf(out, "🏢 %s %s...\n", ui.Bold(out, "Fetching repositories for organization:"), ui.Cyan(out, summaryOrg))
		orgRepos, err := client.ListOrgRepositories(cmd.Context(), summaryOrg)
		if err != nil {
			return fmt.Errorf("failed to list organization repositories: %w", err)
		}
		selectedRepos = nil
		for _, r := range orgRepos {
			selectedRepos = append(selectedRepos, r.FullName)
		}
		fmt.Fprintf(out, "✅ Found %d repositories in %s\n\n", len(selectedRepos), summaryOrg)
	} else if len(selectedRepos) > 0 && !summaryPickRepos && ui.IsInteractiveTerminal(cmd.InOrStdin()) {
		pick, err := askWhetherPickRepos(cmd, selectedRepos)
		if err != nil {
			return err
		}
		if pick {
			summaryPickRepos = true
		}
	}

	if len(selectedRepos) == 0 || summaryPickRepos {
		var err error
		selectedRepos, err = selectRepositories(cmd, repos)
		if err != nil {
			if errors.Is(err, ui.ErrSelectionCancelled) {
				fmt.Fprintln(out, ui.Yellow(out, "Repository selection cancelled."))
				return nil
			}
			return err
		}
		viper.Set(getProfileKey(profileName, "repositories"), selectedRepos)
		_ = saveConfig()
	} else {
		fmt.Fprintf(out, "%s %s\n\n", ui.Bold(out, "Using saved repositories:"), ui.Gray(out, strings.Join(selectedRepos, ", ")))
	}

	resolvedBranches, branchWarnings, err := resolveSummaryBranches(cmd, client, selectedRepos)
	if err != nil {
		if errors.Is(err, ui.ErrSelectionCancelled) {
			fmt.Fprintln(out, ui.Yellow(out, "Branch selection cancelled."))
			return nil
		}
		return err
	}

	user, err := client.GetAuthenticatedUser(cmd.Context())
	if err != nil {
		if errors.Is(err, githubapi.ErrUnauthorized) {
			return fmt.Errorf("stored token is invalid or expired. run `github-work-summary login` again")
		}
		return err
	}

	authorFilter := user.Login
	if summaryOrg != "" {
		authorFilter = "" // Fetch everyone's work for org summary
	}

	windowEnd := time.Now()
	if summaryUntil != "" {
		if parsedUntil, err := summary.ParseFlexibleTime(summaryUntil, time.Now()); err == nil {
			windowEnd = parsedUntil
		}
	}

	windowStart := windowEnd.Add(-defaultSummaryWindow)
	if summarySince != "" {
		if parsedSince, err := summary.ParseFlexibleTime(summarySince, windowEnd); err == nil {
			windowStart = parsedSince
		}
	} else if summaryDays > 0 {
		windowStart = windowEnd.AddDate(0, 0, -summaryDays)
	} else if summaryDuration != "" {
		if d, err := summary.ParseFlexibleDuration(summaryDuration); err == nil {
			windowStart = windowEnd.Add(-d)
		}
	}

	repoCommits, repoPulls, branchStatus, warnings, err := summary.FetchWorkData(cmd.Context(), client, selectedRepos, authorFilter, windowStart, resolvedBranches, summarySkipPRs)
	if err != nil {
		return err
	}

	var allCommits []githubapi.Commit
	var allPulls []githubapi.PullRequest
	for _, commits := range repoCommits {
		allCommits = append(allCommits, commits...)
	}
	for _, pulls := range repoPulls {
		allPulls = append(allPulls, pulls...)
	}

	// Ticket Extraction
	extractTicketsFromCommits(allCommits)

	report := summary.BuildReport(allCommits, allPulls, windowStart, windowEnd)
	allWarnings := append([]string(nil), branchWarnings...)
	allWarnings = append(allWarnings, warnings...)

	// Fetch Ticket Metadata
	fetchTicketMetadata(cmd.Context(), &report)

	if report.TotalCommits == 0 {
		fallbackStart := windowEnd.Add(-fallbackSummaryWindow)
		fbCommitsMap, fbPullsMap, fbStatus, fbWarnings, err := summary.FetchWorkData(cmd.Context(), client, selectedRepos, user.Login, fallbackStart, resolvedBranches, summarySkipPRs)
		if err == nil {
			var fbCommits []githubapi.Commit
			var fbPulls []githubapi.PullRequest
			for _, c := range fbCommitsMap {
				fbCommits = append(fbCommits, c...)
			}
			for _, p := range fbPullsMap {
				fbPulls = append(fbPulls, p...)
			}

			fbReport := summary.BuildReport(fbCommits, fbPulls, fallbackStart, windowEnd)
			if fbReport.TotalCommits > 0 {
				fmt.Fprintf(out, "%s\n\n", ui.Yellow(out, fmt.Sprintf("No commits found in the last 24 hours. Showing recent commits from the last %d days instead.", int(fallbackSummaryWindow.Hours()/24))))
				report = fbReport
				branchStatus = fbStatus
				allWarnings = append(allWarnings, fbWarnings...)
			}
		}
	}

	// AI Summarization
	if summaryAI {
		provider := viper.GetString("ai_provider")
		if provider == "" {
			provider = "gemini"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var aiProvider ai.Provider
		var err error

		switch provider {
		case "gemini":
			key, _ := getAIKey("gemini")
			if key == "" {
				fmt.Fprintln(out, ui.Red(out, "Error: Google AI API Key not found. Run `gws ai-login`."))
				return nil
			}
			aiProvider, err = ai.NewGeminiProvider(ctx, key)
		case "anthropic", "claude":
			key, _ := getAIKey("anthropic")
			if key == "" {
				fmt.Fprintln(out, ui.Red(out, "Error: Anthropic API Key not found. Run `gws ai-login --provider anthropic`."))
				return nil
			}
			aiProvider = ai.NewAnthropicProvider(key)
		case "ollama":
			aiProvider = ai.NewOllamaProvider()
		default:
			fmt.Fprintf(out, ui.Red(out, "Error: Unsupported AI provider: %s\n"), provider)
			return nil
		}

		if err != nil {
			fmt.Fprintf(out, ui.Red(out, "Error initializing AI provider: %v\n"), err)
			return nil
		}

		fmt.Fprint(out, ui.Gray(out, fmt.Sprintf("Generating AI insights via %s... ", provider)))

		var summaryText string
		persona := summaryPersona
		if summaryAudit {
			persona = "audit"
		}

		if report.WindowEnd.Sub(report.WindowStart) > 25*time.Hour && persona == "default" {
			summaryText, err = aiProvider.GenerateTrendAnalysis(ctx, report)
		} else {
			summaryText, err = aiProvider.SummarizeWithPersona(ctx, report, persona)
		}

		if err != nil {
			fmt.Fprintln(out, ui.Red(out, "failed."))
			fmt.Fprintf(out, "%s %v\n", ui.Red(out, "AI error:"), err)
		} else {
			fmt.Fprintln(out, ui.Green(out, "Done."))
			report.AISummary = summaryText
		}
	}

	// Interactive Dashboard
	if summaryInteractive && (report.TotalCommits > 0 || report.TotalPRs > 0) {
		m := tui.NewMainModel(report, func(platform string, r summary.Report) error {
			notifier, err := getNotifier(platform)
			if err != nil {
				return err
			}
			return notifier.Send(context.Background(), r)
		})
		p := tea.NewProgram(m, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(out, "%s %v\n", ui.Red(out, "Dashboard error:"), err)
		} else {
			if updatedModel, ok := finalModel.(tui.MainModel); ok && updatedModel.ExitReport != nil {
				report = *updatedModel.ExitReport
			}
		}
		return nil
	}

	// Direct Sharing
	if summaryShare != "" && (report.TotalCommits > 0 || report.TotalPRs > 0) {
		fmt.Fprintf(out, "%s %s... ", ui.Gray(out, "Sharing summary to"), ui.Cyan(out, summaryShare))
		notifier, err := getNotifier(summaryShare)
		if err != nil {
			fmt.Fprintf(out, "%s %v\n", ui.Red(out, "error:"), err)
		} else {
			if err := notifier.Send(cmd.Context(), report); err != nil {
				fmt.Fprintf(out, "%s %v\n", ui.Red(out, "delivery failed:"), err)
			} else {
				fmt.Fprintln(out, ui.Green(out, "Done."))
			}
		}
	}

	renderBranchStatus(out, branchStatus)
	renderBranchFilter(out, resolvedBranches)

	var output string
	var errExport error
	if summaryJSON {
		var b []byte
		b, errExport = report.ToJSON()
		output = string(b)
	} else if summaryHTML {
		output, errExport = report.ToHTML()
	} else if summaryMarkdown {
		output = report.ToMarkdown()
	} else if summaryOutputFile != "" {
		// Auto-detect format from extension
		ext := strings.ToLower(filepath.Ext(summaryOutputFile))
		switch ext {
		case ".json":
			var b []byte
			b, errExport = report.ToJSON()
			output = string(b)
		case ".html", ".htm":
			output, errExport = report.ToHTML()
		default:
			output = report.ToMarkdown()
		}
	}

	if errExport != nil {
		return fmt.Errorf("failed to generate export: %w", errExport)
	}

	if output == "" {
		summary.Render(out, report)
	} else {
		if summaryOutputFile != "" {
			if err := os.WriteFile(summaryOutputFile, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Fprintf(out, "\n%s %s\n", ui.Green(out, "Successfully exported summary to:"), ui.Bold(out, summaryOutputFile))
		} else {
			fmt.Fprintln(out, output)
		}
	}

	if len(allWarnings) > 0 {
		fmt.Fprintln(out, ui.Bold(out, ui.Yellow(out, "Warnings:")))
		for _, warning := range allWarnings {
			fmt.Fprintf(out, "%s %s\n", ui.Red(out, "•"), warning)
		}
	}
	return nil
}

func resolveSummaryBranches(cmd *cobra.Command, client githubapi.GitHubClient, selectedRepos []string) ([]string, []string, error) {
	branches := sanitizeBranches(summaryBranches)
	if len(branches) == 0 {
		profileName := getActiveProfileName()
		branches = viper.GetStringSlice(getProfileKey(profileName, "branches"))
	}
	if len(branches) > 0 {
		return branches, nil, nil
	}
	in := cmd.InOrStdin()
	if !ui.IsInteractiveTerminal(in) {
		return nil, nil, nil
	}
	if !summaryChooseBranch {
		choose, err := askWhetherChooseBranch(cmd)
		if err != nil {
			return nil, nil, err
		}
		if !choose {
			return nil, nil, nil
		}
	}
	branchRepoCount, warnings, err := summary.FetchBranchesAcrossRepos(cmd.Context(), client, selectedRepos)
	if err != nil {
		return nil, nil, err
	}
	selected, err := selectBranches(cmd, branchRepoCount)
	if err != nil {
		return nil, nil, err
	}
	profileName := getActiveProfileName()
	viper.Set(getProfileKey(profileName, "branches"), selected)
	_ = saveConfig()
	return selected, warnings, nil
}

func askWhetherChooseBranch(cmd *cobra.Command) (bool, error) {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()
	fmt.Fprintln(out, ui.Gray(out, "Branch filter: press Enter for all branches, or type 'b' then Enter to choose branch(es)."))
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	choice := strings.TrimSpace(strings.ToLower(line))
	return choice == "b" || choice == "branch", nil
}

func askWhetherPickRepos(cmd *cobra.Command, currentRepos []string) (bool, error) {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()
	fmt.Fprintf(out, "%s Press Enter to use %d saved repositories, or type 'p' then Enter to pick different ones.\n", ui.Gray(out, "Repos:"), len(currentRepos))
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	choice := strings.TrimSpace(strings.ToLower(line))
	return choice == "p" || choice == "pick", nil
}

func loadGitHubClientFromKeychain() (githubapi.GitHubClient, error) {
	store := auth.NewKeyringStore(auth.DefaultServiceName, auth.DefaultTokenAccount)
	token, err := store.GetToken()
	if err != nil {
		return nil, fmt.Errorf("unable to read GitHub token: %w. run `github-work-summary login` first", err)
	}
	return githubapi.NewClient(token)
}

func selectRepositories(cmd *cobra.Command, repos []githubapi.Repository) ([]string, error) {
	options := make([]ui.SelectOption, 0, len(repos))
	for _, repo := range repos {
		label := repo.FullName
		if repo.Private {
			label += " (private)"
		}
		options = append(options, ui.SelectOption{Label: label, Value: repo.FullName})
	}
	selected, err := ui.MultiSelectCheckboxes(cmd.InOrStdin(), cmd.OutOrStdout(), "Select repositories:", options)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(selected))
	for _, item := range selected {
		result = append(result, item.Value)
	}
	return result, nil
}

func renderBranchStatus(out io.Writer, statusByRepo map[string]summary.RepoBranchStatus) {
	if len(statusByRepo) == 0 {
		return
	}
	repos := make([]string, 0, len(statusByRepo))
	for r := range statusByRepo {
		repos = append(repos, r)
	}
	sort.Strings(repos)
	fmt.Fprintln(out, ui.Bold(out, ui.Cyan(out, "Branch Activity:")))
	for _, r := range repos {
		s := statusByRepo[r]
		if len(s.Scanned) == 0 {
			fmt.Fprintf(out, " • %s: %s\n", ui.Bold(out, r), ui.Gray(out, "no branches scanned"))
			continue
		}
		fmt.Fprintf(out, " • %s: %s\n", ui.Bold(out, r), ui.Gray(out, joinBranchNamesWithLimit(s.Scanned, 6)))
	}
	fmt.Fprintln(out)
}

func renderBranchFilter(out io.Writer, branches []string) {
	if len(branches) == 0 {
		fmt.Fprintf(out, "%s %s\n\n", ui.Bold(out, "Branch Filter:"), ui.Gray(out, "all branches"))
		return
	}
	fmt.Fprintf(out, "%s %s\n\n", ui.Bold(out, "Branch Filter:"), ui.Cyan(out, strings.Join(branches, ", ")))
}

func extractTicketsFromCommits(commits []githubapi.Commit) {
	for i := range commits {
		commits[i].Tickets = tickets.ExtractTicketIDs(commits[i].Message)
	}
}

func fetchTicketMetadata(ctx context.Context, r *summary.Report) {
	uniqueIDs := make(map[string]struct{})
	for _, repo := range r.Repositories {
		allC := append(repo.Features, repo.BugFixes...)
		allC = append(allC, repo.Maintenance...)
		allC = append(allC, repo.Other...)
		for _, c := range allC {
			for _, id := range c.Tickets {
				uniqueIDs[id] = struct{}{}
			}
		}
	}

	if len(uniqueIDs) == 0 {
		return
	}

	// Initialize Providers
	var provs []tickets.Provider

	// Jira
	jiraDomain := viper.GetString("jira_domain")
	jiraEmail := viper.GetString("jira_email")
	if jiraDomain != "" && jiraEmail != "" {
		store := auth.NewKeyringStore("gws-jira", jiraEmail)
		if token, _ := store.GetToken(); token != "" {
			provs = append(provs, tickets.NewJiraProvider(jiraDomain, jiraEmail, token))
		}
	}

	// Linear
	store := auth.NewKeyringStore("gws-linear", "default")
	if token, _ := store.GetToken(); token != "" {
		provs = append(provs, tickets.NewLinearProvider(token))
	}

	if len(provs) == 0 {
		return
	}

	for id := range uniqueIDs {
		for _, p := range provs {
			if p.CanHandle(id) {
				t, err := p.FetchTicket(ctx, id)
				if err == nil {
					r.TicketInfo = append(r.TicketInfo, summary.Ticket{
						ID:     t.ID,
						Title:  t.Title,
						URL:    t.URL,
						Status: t.Status,
					})
					break // Found it
				}
			}
		}
	}
}

func joinBranchNamesWithLimit(branches []string, limit int) string {
	if len(branches) == 0 {
		return "none"
	}
	if len(branches) <= limit {
		return strings.Join(branches, ", ")
	}
	return fmt.Sprintf("%s (+%d more)", strings.Join(branches[:limit], ", "), len(branches)-limit)
}

func sanitizeBranches(branches []string) []string {
	return summary.SanitizeAndSortBranches(branches)
}

func selectBranches(cmd *cobra.Command, branchRepoCount map[string]int) ([]string, error) {
	var names []string
	for n := range branchRepoCount {
		names = append(names, n)
	}
	sort.Strings(names)
	var options []ui.SelectOption
	for _, n := range names {
		options = append(options, ui.SelectOption{Label: fmt.Sprintf("%s (%d repos)", n, branchRepoCount[n]), Value: n})
	}
	selected, err := ui.MultiSelectCheckboxes(cmd.InOrStdin(), cmd.OutOrStdout(), "Select branches:", options)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, s := range selected {
		res = append(res, s.Value)
	}
	return res, nil
}
