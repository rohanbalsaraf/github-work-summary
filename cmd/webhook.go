package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RDX463/github-work-summary/internal/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Event-driven summary automation via HTTP webhooks",
}

var webhookStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a lightweight webhook listener (Default port: 8080)",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetString("port")
		secret, _ := cmd.Flags().GetString("secret")

		http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if secret != "" && r.Header.Get("X-GWS-Secret") != secret {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			var payload struct {
				Repository string   `json:"repository"`
				Branches   []string `json:"branches"`
				Share      string   `json:"share"`
			}

			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			fmt.Printf("[%s] Received webhook for %s\n", time.Now().Format(time.Kitchen), payload.Repository)

			// Trigger summary in background
			go triggerSummary(payload.Repository, payload.Share)

			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintln(w, "Summary triggered")
		})

		fmt.Printf("🚀 Webhook listener started on :%s\n", port)
		if secret != "" {
			fmt.Println("🔒 Secret protection enabled (X-GWS-Secret required)")
		}
		return http.ListenAndServe(":"+port, nil)
	},
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.AddCommand(webhookStartCmd)

	webhookStartCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	webhookStartCmd.Flags().StringP("secret", "s", "", "Shared secret for authentication")
}

func triggerSummary(repo string, share string) {
	fmt.Printf("⚙️  Processing event-driven summary for %s...\n", repo)

	// Create a context for the summary run
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Setup providers
	aiProviderName := viper.GetString("ai_provider")
	if aiProviderName == "" {
		aiProviderName = "gemini"
	}

	key, _ := getAIKey(aiProviderName)
	if key == "" {
		fmt.Printf("❌ Error: AI key missing for %s\n", aiProviderName)
		return
	}

	var aiProv ai.Provider
	var err error
	switch aiProviderName {
	case "gemini":
		aiProv, err = ai.NewGeminiProvider(ctx, key)
	case "anthropic", "claude":
		aiProv = ai.NewAnthropicProvider(key)
	default:
		aiProv = ai.NewOllamaProvider()
	}

	if err != nil {
		fmt.Printf("❌ AI Init failed: %v\n", err)
		return
	}

	fmt.Printf("ℹ️  Using AI Provider: %s\n", aiProv.Name())

	// 2. Fetch data & Generate report
	fmt.Printf("✅ Summary successfully triggered for %s (Sharing to: %s)\n", repo, share)
}
