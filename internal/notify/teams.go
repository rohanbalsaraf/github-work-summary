package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RDX463/github-work-summary/internal/summary"
)

// TeamsNotifier posts a professional Adaptive Card to a Microsoft Teams Webhook.
type TeamsNotifier struct {
	WebhookURL string
}

func (n *TeamsNotifier) Send(ctx context.Context, report summary.Report) error {
	payload := n.buildTeamsPayload(report)
	return sendPost(ctx, n.WebhookURL, payload)
}

func (n *TeamsNotifier) buildTeamsPayload(report summary.Report) map[string]interface{} {
	title := fmt.Sprintf("🚀 GitHub Work Summary (%s)", report.WindowEnd.Format("Jan 02, 2006"))

	body := []map[string]interface{}{
		map[string]interface{}{
			"type":   "TextBlock",
			"text":   title,
			"size":   "large",
			"weight": "bolder",
		},
		map[string]interface{}{
			"type":     "TextBlock",
			"text":     fmt.Sprintf("Window: %s to %s", report.WindowStart.Format(time.RFC822), report.WindowEnd.Format(time.RFC822)),
			"isSubtle": true,
			"spacing":  "none",
		},
	}

	if report.AISummary != "" {
		body = append(body, map[string]interface{}{
			"type":    "TextBlock",
			"text":    "✨ AI Impact Summary",
			"weight":  "bolder",
			"spacing": "medium",
		})
		body = append(body, map[string]interface{}{
			"type": "TextBlock",
			"text": report.AISummary,
			"wrap": true,
		})
	}

	if len(report.TicketInfo) > 0 {
		var tickets []string
		for _, t := range report.TicketInfo {
			tickets = append(tickets, fmt.Sprintf("- [%s](%s): %s", t.ID, t.URL, t.Title))
		}
		body = append(body, map[string]interface{}{
			"type":    "TextBlock",
			"text":    "🎫 Related Tickets",
			"weight":  "bolder",
			"spacing": "medium",
		})
		body = append(body, map[string]interface{}{
			"type": "TextBlock",
			"text": strings.Join(tickets, "\n"),
			"wrap": true,
		})
	}

	// Repository Activity
	for _, repo := range report.Repositories {
		stats := fmt.Sprintf("%d Commits, %d PRs",
			len(repo.Features)+len(repo.BugFixes)+len(repo.Maintenance)+len(repo.Other),
			len(repo.PullRequests))

		body = append(body, map[string]interface{}{
			"type": "FactSet",
			"facts": []map[string]interface{}{
				map[string]interface{}{"title": "📦 " + repo.Repository, "value": stats},
			},
			"spacing": "small",
		})
	}

	return map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			map[string]interface{}{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body":    body,
				},
			},
		},
	}
}
