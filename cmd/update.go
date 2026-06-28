package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/RDX463/github-work-summary/internal/update"
	"github.com/RDX463/github-work-summary/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates to github-work-summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s %s\n", ui.Bold(out, "Current Version:"), ui.Gray(out, version.Version))
	fmt.Fprint(out, ui.Gray(out, "Checking GitHub for updates... "))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := update.Check(ctx, version.Repo, version.Current())
	if err != nil {
		fmt.Fprintf(out, "%s %v\n", ui.Red(out, "failed:"), err)
		return nil
	}
	fmt.Fprintln(out, ui.Green(out, "Done."))

	if !info.UpdateAvailable {
		fmt.Fprintln(out, ui.Green(out, "\n✓ You are already using the latest version."))
		return nil
	}

	fmt.Fprintf(out, "\n%s %s\n", ui.Bold(out, "New version found:"), ui.Green(out, info.LatestVersion))
	fmt.Fprintln(out, ui.Bold(out, "\nUpgrade via Homebrew:"))
	fmt.Fprintf(out, "  %s\n", ui.Cyan(out, "brew upgrade github-work-summary"))

	return nil
}
