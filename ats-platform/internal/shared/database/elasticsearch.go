package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

// ESConfig holds the configuration for Elasticsearch connection
type ESConfig struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string
	CloudID   string
	// InsecureSkipVerify controls whether to verify the server's certificate
	InsecureSkipVerify bool
}

// ESClient wraps the Elasticsearch client
type ESClient struct {
	Client *elasticsearch.Client
}

// NewESClient creates a new Elasticsearch client
func NewESClient(cfg ESConfig) (*ESClient, error) {
	// Set defaults
	if len(cfg.Addresses) == 0 {
		cfg.Addresses = []string{"http://localhost:9200"}
	}

	// Create Elasticsearch client configuration
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		CloudID:   cfg.CloudID,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.InsecureSkipVerify,
			},
		},
		// Retry on 429 TooManyRequests
		RetryOnStatus: []int{502, 503, 504, 429},
		// Retry up to 3 times
		MaxRetries: 3,
		// Compress requests
		CompressRequestBody: true,
	}

	// Create the client
	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	return &ESClient{Client: client}, nil
}

// Ping tests the Elasticsearch connection
func (e *ESClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := e.Client.Ping(e.Client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to ping elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping returned error: %s", res.Status())
	}

	return nil
}

// Close closes the Elasticsearch client connection
// Note: The go-elasticsearch client doesn't have an explicit Close method
// The HTTP transport's idle connections will be closed automatically
func (e *ESClient) Close() error {
	// No explicit close needed for elasticsearch client
	return nil
}

// GetClient returns the underlying Elasticsearch client
func (e *ESClient) GetClient() *elasticsearch.Client {
	return e.Client
}
