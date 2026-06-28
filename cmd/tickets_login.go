package cmd

import (
	"fmt"
	"strings"

	"github.com/RDX463/github-work-summary/internal/auth"
	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ticketsLoginCmd = &cobra.Command{
	Use:   "tickets-login [jira|linear]",
	Short: "Store API credentials for Jira or Linear",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := strings.ToLower(args[0])
		switch provider {
		case "jira":
			return handleJiraLogin(cmd)
		case "linear":
			return handleLinearLogin(cmd)
		default:
			return fmt.Errorf("unsupported provider: %s", provider)
		}
	},
}

func init() {
	rootCmd.AddCommand(ticketsLoginCmd)
}

func handleJiraLogin(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	fmt.Fprintln(out, ui.Bold(out, "Jira Configuration"))

	domain, err := ui.Prompt(in, out, "Enter your Jira domain (e.g. yourbase.atlassian.net):")
	if err != nil {
		return err
	}

	email, err := ui.Prompt(in, out, "Enter your Atlassian email:")
	if err != nil {
		return err
	}

	token, err := ui.PromptPassword(in, out, "Enter your Jira API Token:")
	if err != nil {
		return err
	}

	// Store domain and email in config
	viper.Set("jira_domain", domain)
	viper.Set("jira_email", email)
	saveConfig()

	// Store token in keychain
	store := auth.NewKeyringStore("gws-jira", email)
	if err := store.SaveToken(token); err != nil {
		return fmt.Errorf("failed to store Jira token: %w", err)
	}

	fmt.Fprintln(out, ui.Green(out, "\nJira credentials saved successfully!"))
	return nil
}

func handleLinearLogin(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	fmt.Fprintln(out, ui.Bold(out, "Linear Configuration"))

	token, err := ui.PromptPassword(in, out, "Enter your Linear API Key:")
	if err != nil {
		return err
	}

	// Store token in keychain
	store := auth.NewKeyringStore("gws-linear", "default")
	if err := store.SaveToken(token); err != nil {
		return fmt.Errorf("failed to store Linear token: %w", err)
	}

	fmt.Fprintln(out, ui.Green(out, "\nLinear credentials saved successfully!"))
	return nil
}

func RunTicketsLogin(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	fmt.Fprintln(out, ui.Bold(out, "Choose Ticket Provider"))
	options := []ui.SelectOption{
		{Label: "Jira", Value: "jira"},
		{Label: "Linear", Value: "linear"},
	}
	selected, err := ui.MultiSelectCheckboxes(in, out, "Select a provider to configure:", options)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return nil
	}

	provider := selected[0].Value
	if provider == "jira" {
		return handleJiraLogin(cmd)
	}
	return handleLinearLogin(cmd)
}
