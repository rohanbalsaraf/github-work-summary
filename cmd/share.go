package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RDX463/github-work-summary/internal/auth"
	"github.com/RDX463/github-work-summary/internal/notify"
	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	slackWebhookService   = "github-work-summary-slack-webhook"
	discordWebhookService = "github-work-summary-discord-webhook"
	teamsWebhookService   = "github-work-summary-teams-webhook"
	notionTokenService    = "github-work-summary-notion-token"
	smtpPassService       = "github-work-summary-smtp-pass"
	webhookAccount        = "url"
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share your work summary with your team",
}

var shareSetupCmd = &cobra.Command{
	Use:   "setup [slack|discord|teams|notion|email]",
	Short: "Configure sharing platforms for work summaries",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("please specify a platform: slack, discord, teams, notion, or email")
		}
		return runShareSetup(cmd, args[0])
	},
}

func init() {
	rootCmd.AddCommand(shareCmd)
	shareCmd.AddCommand(shareSetupCmd)
}

func runShareSetup(cmd *cobra.Command, platform string) error {
	out := cmd.OutOrStdout()

	switch strings.ToLower(platform) {
	case "slack":
		return setupWebhook(out, "Slack", slackWebhookService, "https://api.slack.com/messaging/webhooks")
	case "discord":
		return setupWebhook(out, "Discord", discordWebhookService, "https://support.discord.com/hc/en-us/articles/228383668")
	case "teams":
		return setupWebhook(out, "Teams", teamsWebhookService, "https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook")
	case "notion":
		return setupNotion(out)
	case "email":
		return setupEmail(out)
	default:
		return fmt.Errorf("unsupported platform: %s (use slack, discord, teams, notion, or email)", platform)
	}
}

func setupWebhook(out io.Writer, name, service, helpURL string) error {
	fmt.Fprintf(out, "%s %s\n", ui.Bold(out, "Configuring Integration:"), ui.Cyan(out, name))
	fmt.Fprintf(out, "To get a webhook URL, visit: %s\n\n", ui.Gray(out, helpURL))
	fmt.Fprintf(out, "%s Enter your %s Webhook URL: ", ui.Bold(out, "Setup:"), name)

	reader := bufio.NewReader(os.Stdin)
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	if url == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL")
	}

	store := auth.NewKeyringStore(service, webhookAccount)
	if err := store.SaveToken(url); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n%s %s integration stored securely.\n", ui.Green(out, "✓"), name)
	return nil
}

func setupNotion(out io.Writer) error {
	fmt.Fprintln(out, ui.Bold(out, "Configuring Notion Integration"))
	fmt.Fprintln(out, "1. Create an integration at https://www.notion.so/my-integrations")
	fmt.Fprintln(out, "2. Share your database with the integration.")

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Notion API Token: ")
	token, _ := reader.ReadString('\n')
	fmt.Print("Enter Notion Database/Page ID: ")
	parentID, _ := reader.ReadString('\n')

	token = strings.TrimSpace(token)
	parentID = strings.TrimSpace(parentID)

	if token == "" || parentID == "" {
		return fmt.Errorf("token and parent ID are required")
	}

	viper.Set("notion_parent_id", parentID)
	viper.WriteConfig()

	store := auth.NewKeyringStore(notionTokenService, "default")
	return store.SaveToken(token)
}

func setupEmail(out io.Writer) error {
	fmt.Fprintln(out, ui.Bold(out, "Configuring Email (SMTP) Integration"))
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("SMTP Host (e.g. smtp.gmail.com): ")
	host, _ := reader.ReadString('\n')
	fmt.Print("SMTP Port (e.g. 587): ")
	port, _ := reader.ReadString('\n')
	fmt.Print("SMTP User (your email): ")
	user, _ := reader.ReadString('\n')
	fmt.Print("Recipient Email: ")
	to, _ := reader.ReadString('\n')
	fmt.Print("SMTP Password: ")
	pass, _ := reader.ReadString('\n')

	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	user = strings.TrimSpace(user)
	to = strings.TrimSpace(to)
	pass = strings.TrimSpace(pass)

	viper.Set("smtp_host", host)
	viper.Set("smtp_port", port)
	viper.Set("smtp_user", user)
	viper.Set("smtp_to", to)
	viper.WriteConfig()

	store := auth.NewKeyringStore(smtpPassService, "default")
	return store.SaveToken(pass)
}

func getWebhook(platform string) (string, error) {
	var serviceName string
	switch strings.ToLower(platform) {
	case "slack":
		serviceName = slackWebhookService
	case "discord":
		serviceName = discordWebhookService
	case "teams":
		serviceName = teamsWebhookService
	default:
		return "", fmt.Errorf("unsupported webhook platform: %s", platform)
	}

	store := auth.NewKeyringStore(serviceName, webhookAccount)
	token, err := store.GetToken()
	if err != nil || token == "" {
		return "", fmt.Errorf("webhook not configured. Run `gws share setup %s` first", strings.ToLower(platform))
	}
	return token, nil
}

func getNotifier(platform string) (notify.Notifier, error) {
	plat := strings.ToLower(platform)
	switch plat {
	case "slack", "discord", "teams":
		url, err := getWebhook(plat)
		if err != nil {
			return nil, err
		}
		if plat == "slack" {
			return &notify.SlackNotifier{WebhookURL: url}, nil
		}
		if plat == "discord" {
			return &notify.DiscordNotifier{WebhookURL: url}, nil
		}
		return &notify.TeamsNotifier{WebhookURL: url}, nil
	case "notion":
		parentID := viper.GetString("notion_parent_id")
		if parentID == "" {
			return nil, fmt.Errorf("notion database ID not configured. run `gws share setup notion`")
		}
		store := auth.NewKeyringStore(notionTokenService, "default")
		token, _ := store.GetToken()
		if token == "" {
			return nil, fmt.Errorf("notion token not found. run `gws share setup notion`")
		}
		return &notify.NotionNotifier{Token: token, ParentID: parentID}, nil
	case "email":
		host := viper.GetString("smtp_host")
		port := viper.GetString("smtp_port")
		user := viper.GetString("smtp_user")
		to := viper.GetString("smtp_to")
		if host == "" || user == "" {
			return nil, fmt.Errorf("email not configured. run `gws share setup email`")
		}
		store := auth.NewKeyringStore(smtpPassService, "default")
		pass, _ := store.GetToken()
		if pass == "" {
			return nil, fmt.Errorf("email password not found. run `gws share setup email`")
		}
		return &notify.EmailNotifier{Host: host, Port: port, User: user, Password: pass, To: to}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}
