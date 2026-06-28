package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/RDX463/github-work-summary/internal/ai"
	githubapi "github.com/RDX463/github-work-summary/internal/github"
	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"strings"
)

var (
	prBaseFlag  string
	prDraftFlag bool
	prTitleFlag string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "AI-powered Pull Request automation",
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Draft a professional PR for your current branch on GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunPRCreate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prCreateCmd)

	prCreateCmd.Flags().StringVarP(&prBaseFlag, "base", "b", "main", "Base branch for the PR (e.g. main, dev)")
	prCreateCmd.Flags().BoolVarP(&prDraftFlag, "draft", "d", true, "Create the PR as a draft")
	prCreateCmd.Flags().StringVarP(&prTitleFlag, "title", "t", "", "Custom PR title (skips AI title generation)")
}

func RunPRCreate(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// 1. Get local context
	fmt.Fprint(out, ui.Gray(out, "Detecting local Git context... "))
	localCtx, err := githubapi.GetLocalGitContext()
	if err != nil {
		fmt.Fprintln(out, ui.Red(out, "failed."))
		return err
	}
	fmt.Fprintln(out, ui.Green(out, "Done."))
	fmt.Fprintf(out, "%s %s (%s)\n", ui.Bold(out, "Target:"), ui.Cyan(out, localCtx.RepoFullName), ui.Gray(out, localCtx.BranchName))

	// 2. Initialize GitHub client
	client, err := loadGitHubClientFromKeychain()
	if err != nil {
		return err
	}

	// 3. Fetch branch delta (commits in head not in base)
	fmt.Fprintf(out, ui.Gray(out, "Comparing %s...%s... "), prBaseFlag, localCtx.BranchName)
	commits, err := client.GetCompareCommits(cmd.Context(), localCtx.RepoFullName, prBaseFlag, localCtx.BranchName)
	if err != nil {
		fmt.Fprintln(out, ui.Red(out, "failed."))
		return err
	}
	if len(commits) == 0 {
		fmt.Fprintln(out, ui.Yellow(out, "no new commits found. PR creation aborted."))
		return nil
	}
	fmt.Fprintln(out, ui.Green(out, "Done."))
	fmt.Fprintf(out, "%s %d new commit(s)\n\n", ui.Bold(out, "Found:"), len(commits))

	// 4. Initialize AI Provider
	providerName := viper.GetString("ai_provider")
	if providerName == "" {
		providerName = "gemini"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var aiProvider ai.Provider
	switch providerName {
	case "gemini":
		key, _ := getAIKey("gemini")
		if key == "" {
			return fmt.Errorf("gemini key missing. run `gws ai-login`")
		}
		aiProvider, err = ai.NewGeminiProvider(ctx, key)
	case "anthropic", "claude":
		key, _ := getAIKey("anthropic")
		if key == "" {
			return fmt.Errorf("anthropic key missing. run `gws ai-login --provider anthropic`")
		}
		aiProvider = ai.NewAnthropicProvider(key)
	case "ollama":
		aiProvider = ai.NewOllamaProvider()
	}

	if err != nil {
		return fmt.Errorf("ai init failed: %w", err)
	}

	// 5. Generate Title & Body
	title := prTitleFlag
	if title == "" {
		fmt.Fprint(out, ui.Gray(out, "Generating AI PR Title... "))
		title, err = aiProvider.GeneratePRTitle(ctx, localCtx.BranchName, commits)
		if err != nil {
			fmt.Fprintln(out, ui.Red(out, "fallback to branch name."))
			title = localCtx.BranchName
		} else {
			fmt.Fprintln(out, ui.Green(out, "Done."))
		}
	}

	fmt.Fprint(out, ui.Gray(out, "Drafting AI PR Description... "))
	body, err := aiProvider.GeneratePRDescription(ctx, localCtx.BranchName, commits)
	if err != nil {
		fmt.Fprintln(out, ui.Red(out, "failed."))
		return err
	}
	fmt.Fprintln(out, ui.Green(out, "Done."))

	// 5.5 AI Risk & Labels
	fmt.Fprint(out, ui.Gray(out, "Analyzing PR Intelligence... "))
	intel, err := aiProvider.GeneratePRIntelligence(ctx, commits)
	if err != nil {
		fmt.Fprintln(out, ui.Yellow(out, "skipped (AI error)."))
	} else {
		fmt.Fprintln(out, ui.Green(out, "Done."))
		fmt.Fprintf(out, "\n%s [%s] %s\n", ui.Bold(out, "Risk Assessment:"), renderRiskLevel(out, intel.RiskLevel), intel.RiskReason)
		if len(intel.RiskAreas) > 0 {
			fmt.Fprintf(out, "%s %s\n", ui.Gray(out, "Scope:"), strings.Join(intel.RiskAreas, ", "))
		}
		if len(intel.SuggestedLabels) > 0 {
			fmt.Fprintf(out, "%s %s\n", ui.Gray(out, "Labels:"), ui.Cyan(out, strings.Join(intel.SuggestedLabels, ", ")))
		}
	}

	// 6. Create PR on GitHub
	fmt.Fprintf(out, ui.Gray(out, "\nCreating %s on GitHub... "), ui.Bold(out, "Draft PR"))
	prNum, prURL, err := client.CreatePullRequest(cmd.Context(), localCtx.RepoFullName, localCtx.BranchName, prBaseFlag, title, body, prDraftFlag)
	if err != nil {
		fmt.Fprintln(out, ui.Red(out, "failed."))
		return err
	}
	fmt.Fprintln(out, ui.Green(out, "Success!"))

	// 7. Apply Labels
	if len(intel.SuggestedLabels) > 0 {
		fmt.Fprint(out, ui.Gray(out, "Applying AI labels... "))
		if err := client.AddLabelsToIssue(cmd.Context(), localCtx.RepoFullName, prNum, intel.SuggestedLabels); err != nil {
			fmt.Fprintln(out, ui.Yellow(out, "failed (optional)."))
		} else {
			fmt.Fprintln(out, ui.Green(out, "Done."))
		}
	}

	fmt.Fprintf(out, "\n%s %s\n", ui.Bold(out, "Pull Request URL:"), ui.Cyan(out, prURL))
	fmt.Fprintln(out, ui.Gray(out, "Review it on GitHub to finalize your submission."))

	return nil
}

func renderRiskLevel(out io.Writer, level string) string {
	switch strings.ToLower(level) {
	case "low":
		return ui.Green(out, level)
	case "medium":
		return ui.Yellow(out, level)
	case "high":
		return ui.Red(out, level)
	default:
		return level
	}
}
