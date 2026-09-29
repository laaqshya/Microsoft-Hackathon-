// Package hindsight adapts the official Hindsight Go client to the pipeline's
// small memory port.
package hindsight

import (
	"context"
	"fmt"
	"strings"
	"time"

	hindsight "github.com/vectorize-io/hindsight/hindsight-clients/go"
)

// Config configures a Hindsight memory bank.
type Config struct {
	URL     string
	BankID  string
	APIKey  string
	Timeout time.Duration
}

// Client is an optional persistent-memory adapter backed by Hindsight.
type Client struct {
	client *hindsight.APIClient
	bankID string
}

// New constructs a client. URL and BankID are required. APIKey is optional for
// self-hosted Hindsight and is sent as a bearer token when supplied.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("hindsight: url is required")
	}
	if cfg.BankID == "" {
		return nil, fmt.Errorf("hindsight: bank id is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	client := hindsight.NewAPIClientWithTimeout(cfg.URL, cfg.APIKey, cfg.Timeout)
	return &Client{client: client, bankID: cfg.BankID}, nil
}

// Recall returns a compact text block suitable for prompt context.
func (c *Client) Recall(ctx context.Context, query string) (string, error) {
	resp, _, err := c.client.MemoryAPI.RecallMemories(ctx, c.bankID).
		RecallRequest(hindsight.RecallRequest{Query: query}).Execute()
	if err != nil {
		return "", fmt.Errorf("hindsight recall: %w", err)
	}
	parts := make([]string, 0, len(resp.Results))
	for _, result := range resp.Results {
		if text := strings.TrimSpace(result.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n- "), nil
}

// Retain stores a sanitized incident outcome asynchronously, so memory
// extraction latency does not hold up the remediation loop.
func (c *Client) Retain(ctx context.Context, content string) error {
	async := true
	_, _, err := c.client.MemoryAPI.RetainMemories(ctx, c.bankID).
		RetainRequest(hindsight.RetainRequest{Items: []hindsight.MemoryItem{{
			Content: hindsight.TextContent(content),
			Context: *hindsight.NewNullableString(hindsight.PtrString("sanitized SRE incident outcome")),
			Tags:    []string{"sre", "incident-outcome"},
		}}, Async: &async}).Execute()
	if err != nil {
		return fmt.Errorf("hindsight retain: %w", err)
	}
	return nil
}
