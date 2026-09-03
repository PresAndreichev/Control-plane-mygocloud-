package client

import (
	"bytes"
	"context"
	"control-plane/cli/config"
	"control-plane/internal/domain"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	http     *http.Client
	endpoint string
}

func New(cfg *config.Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("no API endpoint configured. Run: mygocloud login --endpoint http://localhost:8080")
	}
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		endpoint: cfg.Endpoint,
	}, nil
}

func (c *Client) url(path string) string {
	return c.endpoint + "/api/v1" + path
}

func (c *Client) do(ctx context.Context, method, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}

		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.url(path), bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)

	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Status  int    `json:"status"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return fmt.Errorf("API error [%d %s]: %s", apiErr.Status, apiErr.Code, apiErr.Message)
		}
		return fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func (c *Client) ListApplicationsByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	var apps []domain.Application
	err := c.do(ctx, "GET", "/applications?owner_id="+ownerID.String(), nil, &apps)
	return apps, err
}
