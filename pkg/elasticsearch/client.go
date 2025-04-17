package elasticsearch

import (
	"context"
	"fmt"

	"freescholar-backend/config"

	"github.com/olivere/elastic/v7"
)

// Client is a wrapper around elastic.Client
type Client struct {
	*elastic.Client
}

// NewClient creates a new Elasticsearch client
func NewClient(cfg config.ESConfig) (*Client, error) {
	client, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Test connection
	_, _, err = client.Ping(cfg.URL).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}

	return &Client{client}, nil
}