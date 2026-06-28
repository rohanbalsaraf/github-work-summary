package tickets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var jiraRegex = regexp.MustCompile(`[A-Z][A-Z0-9]+-[0-9]+`)

// JiraProvider fetches ticket metadata from Jira.
type JiraProvider struct {
	BaseURL string // e.g. "https://domain.atlassian.net"
	Email   string
	Token   string
}

type jiraIssueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		Description interface{} `json:"description"` // Jira V3 uses Atlassian Document Format (ADF)
	} `json:"fields"`
}

// NewJiraProvider creates a Jira integration.
func NewJiraProvider(domain, email, token string) *JiraProvider {
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	return &JiraProvider{
		BaseURL: strings.TrimSuffix(domain, "/"),
		Email:   email,
		Token:   token,
	}
}

// FetchTicket retrieves an issue from Jira.
func (p *JiraProvider) FetchTicket(ctx context.Context, id string) (Ticket, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", p.BaseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Ticket{}, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(p.Email + ":" + p.Token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Ticket{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Ticket{}, fmt.Errorf("ticket %s not found in Jira", id)
	}
	if resp.StatusCode != http.StatusOK {
		return Ticket{}, fmt.Errorf("jira API error: %d", resp.StatusCode)
	}

	var result jiraIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Ticket{}, err
	}

	return Ticket{
		ID:       result.Key,
		Title:    result.Fields.Summary,
		Status:   result.Fields.Status.Name,
		URL:      fmt.Sprintf("%s/browse/%s", p.BaseURL, result.Key),
		Platform: "jira",
	}, nil
}

// CanHandle returns true for standard Jira ticket patterns (e.g. ABC-123).
func (p *JiraProvider) CanHandle(id string) bool {
	return jiraRegex.MatchString(id)
}
