package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/RDX463/github-work-summary/internal/update"
	"github.com/RDX463/github-work-summary/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const lastCheckKey = "last_update_check"
const checkInterval = 24 * time.Hour

func maybeNotifyUpdate(cmd *cobra.Command) {
	// Allow disabling in automation or CI.
	if os.Getenv("GWS_NO_UPDATE_CHECK") == "1" {
		return
	}

	// Avoid noise for shell completion generation.
	if cmd != nil && (cmd.Name() == "completion" || cmd.Name() == "update") {
		return
	}

	// Only check once every 24 hours.
	lastCheck := viper.GetTime(lastCheckKey)
	if !lastCheck.IsZero() && time.Since(lastCheck) < checkInterval {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	notice, err := update.Check(ctx, version.Repo, version.Current())

	// Update last check time regardless of error to avoid retry storm if GitHub is down.
	viper.Set(lastCheckKey, time.Now())
	saveConfig()

	if err != nil || notice == nil || !notice.UpdateAvailable {
		return
	}

	out := cmd.ErrOrStderr()
	fmt.Fprintf(
		out,
		"\n%s %s %s\n",
		ui.Bold(out, ui.Yellow(out, "Update available:")),
		ui.Gray(out, notice.CurrentVersion),
		ui.Bold(out, ui.Green(out, notice.LatestVersion)),
	)
	if len(notice.Changes) > 0 {
		fmt.Fprintln(out, ui.Bold(out, ui.Cyan(out, "What's new:")))
		for _, change := range notice.Changes {
			fmt.Fprintf(out, "%s %s\n", ui.Green(out, "•"), change)
		}
	}
	if strings.TrimSpace(notice.URL) != "" {
		fmt.Fprintf(out, "%s %s\n", ui.Bold(out, "Release:"), ui.Cyan(out, notice.URL))
	}
	fmt.Fprintf(
		out,
		"%s %s\n",
		ui.Bold(out, "Update:"),
		ui.Cyan(out, "curl -fsSL https://raw.githubusercontent.com/RDX463/github-work-summary/main/install.sh | bash"),
	)
	fmt.Fprintln(out)
}
