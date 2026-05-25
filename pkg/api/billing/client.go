package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client interface {
	GetSurge(ctx context.Context, riders, drivers int) (float64, error)
	EstimateFare(ctx context.Context, from, to string, riders, drivers int) (FareBreakdown, error)
}

type BillingClient struct {
	baseURL string
	client  *http.Client
}

type SurgeResponse struct {
	Multiplier float64 `json:"multiplier"`
}

type FareBreakdown struct {
	DistanceMiles   float64 `json:"distance_miles"`
	BasePence       int64   `json:"base_pence"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	TotalPence      int64   `json:"total_pence"`
	MinimumApplied  bool    `json:"minimum_applied"`
}

func NewClient(baseURL string, client *http.Client) (*BillingClient, error) {
	if baseURL == "" {
		return nil, errors.New("base URL cannot be empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &BillingClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
	}, nil
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

func (c *BillingClient) EstimateFare(ctx context.Context, from, to string, riders, drivers int) (FareBreakdown, error) {
	reqURL := fmt.Sprintf("%s/fare/estimate?from=%s&to=%s&riders=%d&drivers=%d",
		c.baseURL,
		url.QueryEscape(from),
		url.QueryEscape(to),
		riders,
		drivers,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return FareBreakdown{}, fmt.Errorf("create estimate fare request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return FareBreakdown{}, fmt.Errorf("execute estimate fare request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FareBreakdown{}, parseError(resp)
	}

	var data FareBreakdown
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return FareBreakdown{}, fmt.Errorf("decode estimate fare response: %w", err)
	}

	return data, nil
}

func parseError(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	snippet := string(bytes.TrimSpace(bodyBytes))
	if len(snippet) > 0 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, snippet)
	}
	return fmt.Errorf("request failed with status %d", resp.StatusCode)
}

// Ensure BillingClient implements Client
var _ Client = (*BillingClient)(nil)
