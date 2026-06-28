package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RDX463/github-work-summary/internal/summary"
)

// NotionNotifier creates a new page in a Notion database.
type NotionNotifier struct {
	Token    string
	ParentID string // Database ID or Page ID
}

func (n *NotionNotifier) Send(ctx context.Context, report summary.Report) error {
	payload := n.buildNotionPayload(report)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.notion.com/v1/pages", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+n.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notion api error: status %d", resp.StatusCode)
	}

	return nil
}

func (n *NotionNotifier) buildNotionPayload(report summary.Report) map[string]interface{} {
	title := fmt.Sprintf("Work Summary: %s", report.WindowEnd.Format("Jan 02, 2006"))

	children := []map[string]interface{}{
		{
			"object": "block",
			"type":   "heading_1",
			"heading_1": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": "🚀 GitHub Work Summary"}},
				},
			},
		},
		{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": fmt.Sprintf("Window: %s to %s", report.WindowStart.Format(time.RFC822), report.WindowEnd.Format(time.RFC822))}},
				},
			},
		},
	}

	if report.AISummary != "" {
		children = append(children, map[string]interface{}{
			"object": "block",
			"type":   "heading_2",
			"heading_2": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": "✨ AI Impact Summary"}},
				},
			},
		}, map[string]interface{}{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": report.AISummary}},
				},
			},
		})
	}

	// Grouped activity
	for _, repo := range report.Repositories {
		children = append(children, map[string]interface{}{
			"object": "block",
			"type":   "heading_3",
			"heading_3": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": "📦 " + repo.Repository}},
				},
			},
		})

		stats := fmt.Sprintf("Commits: %d | PRs: %d",
			len(repo.Features)+len(repo.BugFixes)+len(repo.Maintenance)+len(repo.Other),
			len(repo.PullRequests))

		children = append(children, map[string]interface{}{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": stats}},
				},
			},
		})
	}

	return map[string]interface{}{
		"parent": map[string]interface{}{"database_id": n.ParentID},
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []map[string]interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": title}},
				},
			},
		},
		"children": children,
	}
}
