package client

import (
	"context"
	"control-plane/internal/domain"

	"github.com/google/uuid"
)

func (c *Client) CreateApplication(ctx context.Context, name, description, dockerImage string, ownerID uuid.UUID) (*domain.Application, error) {
	req := struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		DockerImage string    `json:"docker_image"`
		OwnerID     uuid.UUID `json:"owner_id"`
	}{Name: name, Description: description, DockerImage: dockerImage, OwnerID: ownerID}

	var app domain.Application
	err := c.do(ctx, "POST", "/applications", req, &app)
	return &app, err
}

func (c *Client) GetApplication(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	var app domain.Application
	err := c.do(ctx, "GET", "/applications/"+id.String(), nil, &app)
	return &app, err
}

func (c *Client) ListApplications(ctx context.Context) ([]domain.Application, error) {
	var apps []domain.Application
	err := c.do(ctx, "GET", "/applications", nil, &apps)
	return apps, err
}
