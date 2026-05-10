package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"medstock/internal/domain"
)

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{pool: pool}
}

func (r *ClientRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Client, error) {
	query := `
		SELECT id, name, api_key, rate_limit
		FROM clients
		WHERE api_key = $1
	`

	var client domain.Client
	err := r.pool.QueryRow(ctx, query, apiKey).Scan(&client.ID, &client.Name, &client.APIKey, &client.RateLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	query := `
		SELECT id, name, api_key, rate_limit
		FROM clients
		WHERE id = $1
	`

	var client domain.Client
	err := r.pool.QueryRow(ctx, query, id).Scan(&client.ID, &client.Name, &client.APIKey, &client.RateLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}