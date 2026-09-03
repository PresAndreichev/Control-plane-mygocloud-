package client

import (
	"context"
	"control-plane/internal/domain"
	"fmt"

	"github.com/google/uuid"
)

func (c *Client) Deploy(ctx context.Context, appID uuid.UUID, version string) (*domain.Deployment, error) {
	req := struct {
		Version string `json:"version"`
	}{Version: version}

	var dep domain.Deployment
	err := c.do(ctx, "POST", "/applications/"+appID.String()+"/deploy", req, &dep)
	return &dep, err
}

func (c *Client) ListDeployments(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	var deps []domain.Deployment
	err := c.do(ctx, "GET", "/applications/"+appID.String()+"/deployments", nil, &deps)
	return deps, err
}

func (c *Client) Rollback(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	var dep domain.Deployment
	err := c.do(ctx, "POST", "/applications/"+appID.String()+"/rollback", nil, &dep)
	return &dep, err
}

func (c *Client) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	path := fmt.Sprintf("/deployments/logs?container_id=%s&tail=%d", containerID, tail)
	var resp struct {
		Logs string `json:"logs"`
	}
	err := c.do(ctx, "GET", path, nil, &resp)
	return resp.Logs, err
}
