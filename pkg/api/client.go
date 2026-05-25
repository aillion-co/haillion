package api

import (
	"net/http"

	"haillion/pkg/api/billing"
	"haillion/pkg/api/matching"
)

type Client struct {
	Billing  billing.Client
	Matching matching.Client
}

type Config struct {
	BillingURL  string
	MatchingURL string
	HTTPClient  *http.Client
}

func NewClient(cfg Config) *Client {
	b, _ := billing.NewClient(cfg.BillingURL, cfg.HTTPClient)
	m, _ := matching.NewClient(cfg.MatchingURL, cfg.HTTPClient)
	return &Client{
		Billing:  b,
		Matching: m,
	}
}
