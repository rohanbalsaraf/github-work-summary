package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/RDX463/github-work-summary/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func runHome(cmd *cobra.Command) error {
	in := cmd.InOrStdin()
	if !ui.IsInteractiveTerminal(in) {
		return cmd.Help()
	}

	inFile, ok := in.(*os.File)
	if !ok {
		return cmd.Help()
	}

	for {
		action, err := ui.RunHomeMenu(inFile, cmd.OutOrStdout(), ui.HomeMenuOptions{
			RepositoryURL: "https://github.com/RDX463/github-work-summary",
			Tagline:       "Summarize your GitHub work from terminal.",
			Version:       version.Current(),
			ActiveProfile: getActiveProfileName(),
		})
		if err != nil {
			return err
		}

		switch action {
		case ui.HomeActionSummary:
			if err := runSummary(cmd); err != nil {
				return err
			}
		case ui.HomeActionPR:
			if err := RunPRCreate(cmd); err != nil {
				return err
			}
		case ui.HomeActionSchedule:
			if err := runSchedule(cmd); err != nil {
				return err
			}
		case ui.HomeActionWebhook:
			if err := runWebhookMenu(cmd); err != nil {
				return err
			}
		case ui.HomeActionRepos:
			if err := runRepos(cmd); err != nil {
				return err
			}
		case ui.HomeActionShare:
			if err := runShareMenu(cmd); err != nil {
				return err
			}
		case ui.HomeActionSwitchProfile:
			if err := runSwitchProfileMenu(cmd); err != nil {
				return err
			}
		case ui.HomeActionLogin:
			if err := runAccountMenu(cmd); err != nil {
				return err
			}
		case ui.HomeActionTickets:
			if err := RunTicketsLogin(cmd); err != nil {
				return err
			}
		case ui.HomeActionAILogin:
			if err := RunAILogin(cmd); err != nil {
				return err
			}
		case ui.HomeActionHelp:
			if err := cmd.Help(); err != nil {
				return err
			}
		case ui.HomeActionVersion:
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s %s\n", ui.Cyan(out, fmt.Sprintf("%s version", cmd.Use)), ui.Bold(out, version.Current()))
		case ui.HomeActionQuit:
			return nil
		default:
			continue
		}

		if err := promptReturnToMenu(inFile, cmd.OutOrStdout()); err != nil {
			return err
		}
	}
}

func promptReturnToMenu(in *os.File, out io.Writer) error {
	fmt.Fprintf(out, "\n%s", ui.Gray(out, "Press Enter to return to menu..."))
	reader := bufio.NewReader(in)
	_, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	fmt.Fprintln(out)
	return nil
}

func runSchedule(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	fmt.Fprintln(out, ui.Bold(out, "Automated Summary Schedule"))
	fmt.Fprint(out, "Enter schedule (HH:MM or 'Monday 09:00'): ")

	reader := bufio.NewReader(in)
	schedStr, _ := reader.ReadString('\n')
	schedStr = strings.TrimSpace(schedStr)

	if schedStr == "" {
		fmt.Fprintln(out, "Cancelled.")
		return nil
	}

	fmt.Fprint(out, "Enter platform (slack/discord): ")
	platform, _ := reader.ReadString('\n')
	platform = strings.TrimSpace(platform)

	if platform == "" {
		fmt.Fprintln(out, "Cancelled.")
		return nil
	}

	// Re-use the command logic by setting flags
	// (Alternatively, I should extract the logic into an internal/schedule/manager.go)
	// For now, call the command via go run or just re-implement briefly or use the cmd flags

	args := []string{schedStr}
	_ = scheduleSetCmd.Flags().Set("share", platform)
	return scheduleSetCmd.RunE(scheduleSetCmd, args)
}

func runWebhookMenu(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, ui.Bold(out, "\n🚀 Starting Webhook Listener..."))
	fmt.Fprintln(out, ui.Gray(out, "(Press Ctrl+C to stop in the future)"))

	// Default to port 8080 if not set
	return webhookStartCmd.RunE(webhookStartCmd, nil)
}

func runShareMenu(_ *cobra.Command) error {
	return shareSetupCmd.RunE(shareSetupCmd, nil)
}

func runAccountMenu(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	fmt.Fprintln(out, ui.Bold(out, "GitHub Account Management"))
	options := []ui.SelectOption{
		{Label: "Login", Value: "login", Desc: "Authenticate and save token"},
		{Label: "Logout", Value: "logout", Desc: "Clear saved credentials"},
	}
	selected, err := ui.MultiSelectCheckboxes(in, out, "Choose an action:", options)
	if err != nil {
		if errors.Is(err, ui.ErrSelectionCancelled) {
			return nil
		}
		return err
	}

	if len(selected) > 0 {
		choice := selected[0].Value
		if choice == "login" {
			return runLogin(cmd)
		}
		return runLogout(cmd)
	}

	return nil
}

func runSwitchProfileMenu(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	names := getProfileNames()
	options := make([]ui.SelectOption, 0, len(names))
	for _, name := range names {
		options = append(options, ui.SelectOption{Label: name, Value: name})
	}

	selected, err := ui.MultiSelectCheckboxes(in, out, "Select a profile to switch to:", options)
	if err != nil {
		if errors.Is(err, ui.ErrSelectionCancelled) {
			return nil
		}
		return err
	}

	if len(selected) > 0 {
		// Just take the first one selected (it's a multi-select UI used as a single-select sort of)
		// Or I could implement a SingleSelect, but MultiSelect is fine for now.
		name := selected[0].Value
		viper.Set(keyActiveProfile, name)
		if err := saveConfig(); err != nil {
			return err
		}
		fmt.Fprintf(out, "\n%s Switched to profile: %s\n", ui.Green(out, "✓"), ui.Bold(out, name))
	}

	return nil
}
