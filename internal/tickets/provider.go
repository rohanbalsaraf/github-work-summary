package tickets

import (
	"context"
)

// Ticket represents common metadata for an issue in Jira or Linear.
type Ticket struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Platform    string `json:"platform"` // jira or linear
}

// Provider defines the interface for fetching issue context.
type Provider interface {
	// FetchTicket returns details for a specific issue ID.
	FetchTicket(ctx context.Context, id string) (Ticket, error)

	// CanHandle returns true if this provider supports the given ID pattern.
	CanHandle(id string) bool
}
