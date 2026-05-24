package api

import (
	"net/http"

	"aillion/pkg/api/billing"
	"aillion/pkg/api/matching"
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
	return &Client{
		Billing:  billing.NewClient(cfg.BillingURL, cfg.HTTPClient),
		Matching: matching.NewClient(cfg.MatchingURL, cfg.HTTPClient),
	}
}
