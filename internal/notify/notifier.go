package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RDX463/github-work-summary/internal/summary"
)

// Notifier defines the interface for sending reports to external platforms.
type Notifier interface {
	Send(ctx context.Context, report summary.Report) error
}

// SlackNotifier posts a rich Block Kit message to a Slack Webhook.
type SlackNotifier struct {
	WebhookURL string
}

func (n *SlackNotifier) Send(ctx context.Context, report summary.Report) error {
	payload := n.buildSlackPayload(report)
	return sendPost(ctx, n.WebhookURL, payload)
}

func (n *SlackNotifier) buildSlackPayload(report summary.Report) map[string]interface{} {
	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]interface{}{
				"type": "plain_text",
				"text": "🚀 GitHub Work Summary",
			},
		},
		{
			"type": "context",
			"elements": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Window:* %s to %s | *Total:* %d Commits, %d PRs",
						report.WindowStart.Format(time.RFC822),
						report.WindowEnd.Format(time.RFC822),
						report.TotalCommits,
						report.TotalPRs),
				},
			},
		},
	}

	if report.AISummary != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*AI IMPACT SUMMARY*\n%s", report.AISummary),
			},
		})
	}

	if len(report.TicketInfo) > 0 {
		var ticketLinks []string
		for _, t := range report.TicketInfo {
			ticketLinks = append(ticketLinks, fmt.Sprintf("<%s|%s>: %s", t.URL, t.ID, t.Title))
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Related Tickets:*\n%s", strings.Join(ticketLinks, "\n")),
			},
		})
	}

	blocks = append(blocks, map[string]interface{}{"type": "divider"})

	for _, repo := range report.Repositories {
		repoText := fmt.Sprintf("*%s* (%d commits, %d PRs)", repo.Repository,
			len(repo.Features)+len(repo.BugFixes)+len(repo.Maintenance)+len(repo.Other),
			len(repo.PullRequests))

		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": repoText,
			},
		})

		// Add details in a compact format
		var details []string
		if len(repo.Features) > 0 {
			details = append(details, fmt.Sprintf("• %d Features", len(repo.Features)))
		}
		if len(repo.BugFixes) > 0 {
			details = append(details, fmt.Sprintf("• %d Bug Fixes", len(repo.BugFixes)))
		}
		if len(repo.PullRequests) > 0 {
			details = append(details, fmt.Sprintf("• %d PRs", len(repo.PullRequests)))
		}

		if len(details) > 0 {
			blocks = append(blocks, map[string]interface{}{
				"type": "context",
				"elements": []map[string]interface{}{
					{
						"type": "mrkdwn",
						"text": strings.Join(details, "  "),
					},
				},
			})
		}
	}

	return map[string]interface{}{"blocks": blocks}
}

// DiscordNotifier posts a rich embed message to a Discord Webhook.
type DiscordNotifier struct {
	WebhookURL string
}

func (n *DiscordNotifier) Send(ctx context.Context, report summary.Report) error {
	payload := n.buildDiscordPayload(report)
	return sendPost(ctx, n.WebhookURL, payload)
}

func (n *DiscordNotifier) buildDiscordPayload(report summary.Report) map[string]interface{} {
	embed := map[string]interface{}{
		"title":       "🚀 GitHub Work Summary",
		"description": fmt.Sprintf("**Window:** %s to %s", report.WindowStart.Format(time.RFC822), report.WindowEnd.Format(time.RFC822)),
		"color":       3447003, // Blueish
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	var fields []map[string]interface{}
	fields = append(fields, map[string]interface{}{
		"name":   "Activity Overview",
		"value":  fmt.Sprintf("✅ **%d** Commits\n🔀 **%d** Pull Requests", report.TotalCommits, report.TotalPRs),
		"inline": true,
	})

	if report.AISummary != "" {
		fields = append(fields, map[string]interface{}{
			"name":  "✨ AI Impact Summary",
			"value": report.AISummary,
		})
	}

	if len(report.TicketInfo) > 0 {
		var ticketText []string
		for _, t := range report.TicketInfo {
			ticketText = append(ticketText, fmt.Sprintf("[%s](%s): %s", t.ID, t.URL, t.Title))
		}
		fields = append(fields, map[string]interface{}{
			"name":  "🎫 Related Tickets",
			"value": strings.Join(ticketText, "\n"),
		})
	}

	for _, repo := range report.Repositories {
		stats := fmt.Sprintf("%d Commits, %d PRs",
			len(repo.Features)+len(repo.BugFixes)+len(repo.Maintenance)+len(repo.Other),
			len(repo.PullRequests))

		fields = append(fields, map[string]interface{}{
			"name":  "📦 " + repo.Repository,
			"value": stats,
		})
	}

	embed["fields"] = fields

	return map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}
}

func sendPost(ctx context.Context, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return nil
}
