package domain

import (
	"context"

	"github.com/google/uuid"
)

type Client struct {
	ID        uuid.UUID
	Name      string
	APIKey    string
	RateLimit int
}

type ClientRepository interface {
	GetByAPIKey(ctx context.Context, apiKey string) (*Client, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Client, error)
}