package tickets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var linearRegex = regexp.MustCompile(`^[A-Z0-9]+-[0-9]+$`)

const linearURL = "https://api.linear.app/graphql"

// LinearProvider fetches ticket metadata from Linear.
type LinearProvider struct {
	Token string
}

type linearGraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type linearGraphQLResponse struct {
	Data struct {
		Issue struct {
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			State      struct {
				Name string `json:"name"`
			} `json:"state"`
			Description string `json:"description"`
			URL         string `json:"url"`
		} `json:"issue"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// NewLinearProvider creates a Linear integration.
func NewLinearProvider(token string) *LinearProvider {
	return &LinearProvider{Token: token}
}

// FetchTicket retrieves an issue from Linear via GraphQL.
func (p *LinearProvider) FetchTicket(ctx context.Context, id string) (Ticket, error) {
	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				identifier
				title
				state {
					name
				}
				description
				url
			}
		}
	`

	reqBody := linearGraphQLRequest{
		Query:     query,
		Variables: map[string]interface{}{"id": id},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Ticket{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, linearURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Ticket{}, err
	}

	req.Header.Set("Authorization", p.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Ticket{}, err
	}
	defer resp.Body.Close()

	var result linearGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Ticket{}, err
	}

	if len(result.Errors) > 0 {
		return Ticket{}, fmt.Errorf("linear API error: %s", result.Errors[0].Message)
	}

	if result.Data.Issue.Identifier == "" {
		return Ticket{}, fmt.Errorf("linear issue %s not found", id)
	}

	return Ticket{
		ID:          result.Data.Issue.Identifier,
		Title:       result.Data.Issue.Title,
		Status:      result.Data.Issue.State.Name,
		Description: result.Data.Issue.Description,
		URL:         result.Data.Issue.URL,
		Platform:    "linear",
	}, nil
}

// CanHandle returns true if the ID matches standard Linear formats.
func (p *LinearProvider) CanHandle(id string) bool {
	return linearRegex.MatchString(id)
}
