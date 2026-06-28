package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/RDX463/github-work-summary/internal/auth"
	"github.com/RDX463/github-work-summary/internal/ui"
	"github.com/spf13/cobra"
)

const (
	googleAIServiceName  = "gws-gemini-api-key"
	anthropicServiceName = "gws-anthropic-api-key"
	googleAIAccountName  = "api-key"
)

var (
	aiProviderFlag string
)

var aiLoginCmd = &cobra.Command{
	Use:   "ai-login",
	Short: "Configure AI API keys securely in OS keychain",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunAILogin(cmd)
	},
}

func init() {
	rootCmd.AddCommand(aiLoginCmd)
	aiLoginCmd.Flags().StringVarP(&aiProviderFlag, "provider", "p", "gemini", "AI provider to login to (gemini, anthropic)")
}

func RunAILogin(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	provider := strings.ToLower(aiProviderFlag)
	var serviceName string
	var url string

	switch provider {
	case "gemini":
		serviceName = googleAIServiceName
		url = "https://aistudio.google.com/app/apikey"
	case "anthropic", "claude":
		provider = "anthropic"
		serviceName = anthropicServiceName
		url = "https://console.anthropic.com/settings/keys"
	case "ollama":
		fmt.Fprintln(out, ui.Green(out, "Ollama is a local provider and does not require an API key."))
		return nil
	default:
		return fmt.Errorf("unsupported provider: %s. Supported: gemini, anthropic", provider)
	}

	fmt.Fprintln(out, ui.Bold(out, fmt.Sprintf("Step 1: Get your %s API Key", strings.Title(provider))))
	fmt.Fprintf(out, "Go to %s and generate a free API key.\n\n", ui.Cyan(out, url))

	fmt.Fprint(out, ui.Bold(out, "Step 2: Enter your API Key: "))

	reader := bufio.NewReader(in)
	key, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)

	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	store := auth.NewKeyringStore(serviceName, googleAIAccountName)
	if err := store.SaveToken(key); err != nil {
		return err
	}

	fmt.Fprintf(out, ui.Green(out, "\n✓ %s API Key stored securely in OS keychain.\n"), strings.Title(provider))
	fmt.Fprintln(out, ui.Gray(out, "You can now use the --ai flag with the summary command."))

	return nil
}

func getAIKey(provider string) (string, error) {
	service := googleAIServiceName
	if provider == "anthropic" {
		service = anthropicServiceName
	}
	store := auth.NewKeyringStore(service, googleAIAccountName)
	return store.GetToken()
}
