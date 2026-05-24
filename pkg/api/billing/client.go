package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	GetSurge(ctx context.Context, riders, drivers int) (float64, error)
}

type BillingClient struct {
	baseURL string
	client  *http.Client
}

type SurgeResponse struct {
	Multiplier float64 `json:"multiplier"`
}

func NewClient(baseURL string, client *http.Client) *BillingClient {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &BillingClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
	}
}

func (c *BillingClient) GetSurge(ctx context.Context, riders, drivers int) (float64, error) {
	reqURL := fmt.Sprintf("%s/surge?riders=%d&drivers=%d", c.baseURL, riders, drivers)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create surge request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("execute surge request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("surge request failed with status: %d", resp.StatusCode)
	}

	var data SurgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode surge response: %w", err)
	}

	return data.Multiplier, nil
}

// Ensure BillingClient implements Client
var _ Client = (*BillingClient)(nil)
