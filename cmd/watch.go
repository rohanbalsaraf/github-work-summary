package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/RDX463/github-work-summary/internal/schedule"
	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start a persistent foreground scheduler",
	RunE: func(cmd *cobra.Command, args []string) error {
		schedStr := viper.GetString("automation.schedule")
		if schedStr == "" {
			return fmt.Errorf("no schedule set. run `gws schedule set \"HH:MM\"` first")
		}

		s, err := schedule.Parse(schedStr)
		if err != nil {
			return err
		}

		platform := viper.GetString("automation.share")
		if platform == "" {
			platform = "slack" // fallback
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "🕰️  %s %s\n", ui.Bold(out, "GWS Scheduler Active:"), ui.Cyan(out, schedStr))
		fmt.Fprintf(out, "📡 %s %s\n", ui.Bold(out, "Sharing to:"), ui.Cyan(out, platform))
		fmt.Fprintln(out, ui.Gray(out, "Press Ctrl+C to stop..."))

		for {
			next := s.NextRun(time.Now())
			wait := time.Until(next)

			fmt.Fprintf(out, "\r%s %s", ui.Gray(out, "Next run in:"), wait.Round(time.Second))

			select {
			case <-time.After(wait):
				fmt.Fprint(out, "\n🚀 Running scheduled summary... ")
				// Trigger the summary
				if err := runAutomatedSummary(cmd.Context(), platform); err != nil {
					fmt.Fprintf(out, "❌ %v\n", err)
				} else {
					fmt.Fprintln(out, "✅ Done.")
				}
			case <-cmd.Context().Done():
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func runAutomatedSummary(_ context.Context, platform string) error {
	// Re-use logic from summary.go but with automated flags
	// For now, call the command directly
	summaryAI = true
	summaryShare = platform
	return runSummary(summaryCmd)
}
