package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	inventory "medstock/internal/domain/inventory"
	"medstock/internal/infrastructure/redis"
	"medstock/pkg/logger"
)

type CatalogService struct {
	inventoryRepo InventoryRepo
	cache         Cache
	log           *logger.Logger
	cacheTTL      time.Duration
}

type CatalogServiceConfig struct {
	InventoryRepo InventoryRepo
	Cache         Cache
	Logger        *logger.Logger
	CacheTTL      time.Duration
}

func NewCatalogService(cfg CatalogServiceConfig) *CatalogService {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	return &CatalogService{
		inventoryRepo: cfg.InventoryRepo,
		cache:         cfg.Cache,
		log:           cfg.Logger,
		cacheTTL:      cfg.CacheTTL,
	}
}

func (s *CatalogService) ListProducts(ctx context.Context, clientID uuid.UUID, page, pageSize int) (*inventory.ProductListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	if s.cache != nil {
		cacheKey := redis.CatalogKey(clientID.String() + ":" + string(rune(page)) + ":" + string(rune(pageSize)))
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			var resp inventory.ProductListResponse
			if err := json.Unmarshal([]byte(cached), &resp); err == nil {
				s.log.Debug().Str("client_id", clientID.String()).Int("page", page).Msg("cache_hit")
				return &resp, nil
			}
		}
	}

	products, total, err := s.inventoryRepo.ListProducts(ctx, clientID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	resp := &inventory.ProductListResponse{
		Products: products,
		Pagination: inventory.Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: (total + pageSize - 1) / pageSize,
		},
	}

	if s.cache != nil {
		data, _ := json.Marshal(resp)
		s.cache.Set(ctx, redis.CatalogKey(clientID.String()), string(data), s.cacheTTL)
	}

	return resp, nil
}

func (s *CatalogService) GetProduct(ctx context.Context, clientID uuid.UUID, sku string) (*inventory.Product, error) {
	if s.cache != nil {
		cacheKey := redis.ProductKey(clientID.String(), sku)
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			var product inventory.Product
			if err := json.Unmarshal([]byte(cached), &product); err == nil {
				return &product, nil
			}
		}
	}

	product, err := s.inventoryRepo.GetProductBySKU(ctx, clientID, sku)
	if err != nil {
		return nil, err
	}

	if product != nil && s.cache != nil {
		data, _ := json.Marshal(product)
		s.cache.Set(ctx, redis.ProductKey(clientID.String(), sku), string(data), s.cacheTTL)
	}

	return product, nil
}

func (s *CatalogService) InvalidateCache(ctx context.Context, clientID uuid.UUID) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Delete(ctx, redis.CatalogKey(clientID.String()))
}