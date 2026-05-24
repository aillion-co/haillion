package matching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Match(ctx context.Context, riderID string, lat, lng float64) (string, int, error)
	UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error
}

type MatchingClient struct {
	baseURL string
	client  *http.Client
}

type MatchRequest struct {
	RiderID string  `json:"rider_id"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type MatchResponse struct {
	DriverID string `json:"driver_id"`
	ETA      int    `json:"eta_seconds"`
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func NewClient(baseURL string, client *http.Client) *MatchingClient {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &MatchingClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
	}
}

func (c *MatchingClient) Match(ctx context.Context, riderID string, lat, lng float64) (string, int, error) {
	reqURL := fmt.Sprintf("%s/match", c.baseURL)

	bodyBytes, err := json.Marshal(MatchRequest{RiderID: riderID, Lat: lat, Lng: lng})
	if err != nil {
		return "", 0, fmt.Errorf("marshal match request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("create match request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("execute match request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("match request failed with status: %d", resp.StatusCode)
	}

	var data MatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", 0, fmt.Errorf("decode match response: %w", err)
	}

	return data.DriverID, data.ETA, nil
}

func (c *MatchingClient) UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error {
	reqURL := fmt.Sprintf("%s/drivers/%s/location", c.baseURL, driverID)

	bodyBytes, err := json.Marshal(UpdateLocationRequest{Lat: lat, Lng: lng})
	if err != nil {
		return fmt.Errorf("marshal location request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create location request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute location request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("location request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Ensure MatchingClient implements Client
var _ Client = (*MatchingClient)(nil)
