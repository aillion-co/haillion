package matching

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Match(ctx context.Context, riderID string, lat, lng float64) (string, int, error)
	UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error
	RequestRide(ctx context.Context, riderID string, postcode string) error
	CancelRide(ctx context.Context, riderID string) error
	NearbyRiders(ctx context.Context, driverID string, radiusMeters float64) ([]NearbyRider, error)
	NearbyDrivers(ctx context.Context, riderID string, radiusMeters float64) ([]NearbyDriver, error)
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

type RiderRequest struct {
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
	Postcode string   `json:"postcode,omitempty"`
}

type NearbyRider struct {
	RiderID   string  `json:"rider_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m"`
}

type NearbyRidersResponse struct {
	Riders []NearbyRider `json:"riders"`
}

type NearbyDriver struct {
	DriverID  string  `json:"driver_id"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m"`
}

type NearbyDriversResponse struct {
	Drivers []NearbyDriver `json:"drivers"`
}

func NewClient(baseURL string, client *http.Client) (*MatchingClient, error) {
	if baseURL == "" {
		return nil, errors.New("base URL cannot be empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &MatchingClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  client,
	}, nil
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

func (c *MatchingClient) RequestRide(ctx context.Context, riderID string, postcode string) error {
	reqURL := fmt.Sprintf("%s/riders/%s/request", c.baseURL, riderID)

	bodyBytes, err := json.Marshal(RiderRequest{Postcode: postcode})
	if err != nil {
		return fmt.Errorf("marshal rider request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create rider request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute rider request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(resp)
	}

	return nil
}

func (c *MatchingClient) CancelRide(ctx context.Context, riderID string) error {
	reqURL := fmt.Sprintf("%s/riders/%s/request", c.baseURL, riderID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, nil)
	if err != nil {
		return fmt.Errorf("create cancel request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute cancel request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(resp)
	}

	return nil
}

func (c *MatchingClient) NearbyRiders(ctx context.Context, driverID string, radiusMeters float64) ([]NearbyRider, error) {
	reqURL := fmt.Sprintf("%s/drivers/%s/nearby-riders?radius_m=%f", c.baseURL, driverID, radiusMeters)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create nearby riders request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute nearby riders request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(resp)
	}

	var data NearbyRidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode nearby riders response: %w", err)
	}

	return data.Riders, nil
}

func (c *MatchingClient) NearbyDrivers(ctx context.Context, riderID string, radiusMeters float64) ([]NearbyDriver, error) {
	reqURL := fmt.Sprintf("%s/riders/%s/nearby-drivers?radius_m=%f", c.baseURL, riderID, radiusMeters)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create nearby drivers request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute nearby drivers request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(resp)
	}

	var data NearbyDriversResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode nearby drivers response: %w", err)
	}

	return data.Drivers, nil
}

func parseError(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	snippet := string(bytes.TrimSpace(bodyBytes))
	if len(snippet) > 0 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, snippet)
	}
	return fmt.Errorf("request failed with status %d", resp.StatusCode)
}

// Ensure MatchingClient implements Client
var _ Client = (*MatchingClient)(nil)
