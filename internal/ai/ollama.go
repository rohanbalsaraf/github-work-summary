package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	githubapi "github.com/RDX463/github-work-summary/internal/github"
	"github.com/RDX463/github-work-summary/internal/summary"
)

const (
	ollamaURL = "http://localhost:11434/api/generate"
)

// OllamaProvider implements the Provider interface for local LLM usage via Ollama.
type OllamaProvider struct {
	model string
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Raw    bool   `json:"raw"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

// NewOllamaProvider creates a new local Ollama provider.
func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{
		model: "llama3", // Standard capable local model
	}
}

// Summarize generates a summary via local Ollama.
func (p *OllamaProvider) Summarize(ctx context.Context, report summary.Report) (string, error) {
	prompt := BuildReportPrompt(report)
	return p.GenerateRaw(ctx, prompt)
}

// GeneratePRDescription drafts a PR body via Ollama.
func (p *OllamaProvider) GeneratePRDescription(ctx context.Context, branchName string, commits []githubapi.Commit) (string, error) {
	prompt := BuildPRPrompt(branchName, commits)
	return p.GenerateRaw(ctx, prompt)
}

// GeneratePRTitle drafts a PR title via Ollama.
func (p *OllamaProvider) GeneratePRTitle(ctx context.Context, branchName string, commits []githubapi.Commit) (string, error) {
	prompt := BuildPRTitlePrompt(branchName, commits)
	return p.GenerateRaw(ctx, prompt)
}

// GeneratePRIntelligence performs a risk assessment and suggests labels using local Ollama.
func (p *OllamaProvider) GeneratePRIntelligence(ctx context.Context, commits []githubapi.Commit) (PRIntelligence, error) {
	prompt := BuildPRIntelligencePrompt(commits)
	raw, err := p.GenerateRaw(ctx, prompt)
	if err != nil {
		return PRIntelligence{}, err
	}

	// Clean JSON if the model wrapped it in markdown
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var intel PRIntelligence
	if err := json.Unmarshal([]byte(raw), &intel); err != nil {
		return PRIntelligence{}, fmt.Errorf("failed to parse PR intelligence JSON: %w", err)
	}

	return intel, nil
}

func (p *OllamaProvider) GenerateRaw(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
		Raw:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed (is it running?): %w", err)
	}
	defer resp.Body.Close()

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}

	return result.Response, nil
}

// GenerateTrendAnalysis summarizes activity over range using Ollama.
func (p *OllamaProvider) GenerateTrendAnalysis(ctx context.Context, report summary.Report) (string, error) {
	prompt := BuildTrendPrompt(report)
	return p.GenerateRaw(ctx, prompt)
}

// SummarizeWithPersona generates a summary using local Ollama tailored to a persona.
func (p *OllamaProvider) SummarizeWithPersona(ctx context.Context, report summary.Report, persona string) (string, error) {
	prompt := getPromptForPersona(report, persona)
	return p.GenerateRaw(ctx, prompt)
}

// Name returns the provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}
