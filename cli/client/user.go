package client

import (
	"context"

	"github.com/google/uuid"

	"control-plane/internal/domain"
)

func (c *Client) CreateUser(ctx context.Context, email, name string) (*domain.User, error) {
	req := struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{Email: email, Name: name}

	var user domain.User
	err := c.do(ctx, "POST", "/users", req, &user)
	return &user, err
}

func (c *Client) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := c.do(ctx, "GET", "/users/"+id.String(), nil, &user)
	return &user, err
}
