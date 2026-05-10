package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	inventory "medstock/internal/domain/inventory"
	upload "medstock/internal/domain/upload"
	"medstock/pkg/logger"
)

type Processor struct {
	pool          *Pool
	dbPool        *pgxpool.Pool
	uploadRepo    UploadRepo
	inventoryRepo InventoryRepo
	cache         Cache
	log           *logger.Logger
	batchSize     int
}

type UploadRepo interface {
	UpdateProgress(ctx context.Context, id uuid.UUID, processed, failed int) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status upload.UploadStatus, errSummary *string) error
	UpdateBatchProgress(ctx context.Context, id uuid.UUID, processed, failed int) error
	UpdateBatchStatus(ctx context.Context, id uuid.UUID, status upload.BatchStatus, errDetail *string) error
}

type InventoryRepo interface {
	UpsertProduct(ctx context.Context, tx pgx.Tx, p *inventory.Product) error
	UpsertInventory(ctx context.Context, tx pgx.Tx, productID uuid.UUID, inv *inventory.Inventory) error
	UpsertPricing(ctx context.Context, tx pgx.Tx, productID uuid.UUID, pr *inventory.Pricing) error
	RecordStockEvent(ctx context.Context, tx pgx.Tx, event *inventory.StockEvent) error
}

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

type ProcessorConfig struct {
	Pool          *Pool
	DBPool        *pgxpool.Pool
	UploadRepo    UploadRepo
	InventoryRepo InventoryRepo
	Cache         Cache
	Logger        *logger.Logger
	BatchSize     int
}

func NewProcessor(cfg ProcessorConfig) *Processor {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	return &Processor{
		pool:          cfg.Pool,
		dbPool:        cfg.DBPool,
		uploadRepo:    cfg.UploadRepo,
		inventoryRepo: cfg.InventoryRepo,
		cache:         cfg.Cache,
		log:           cfg.Logger,
		batchSize:     cfg.BatchSize,
	}
}

func (p *Processor) ProcessChunks(clientID uuid.UUID, items []inventory.BulkItemInput, uploadID uuid.UUID, onProgress func(processed, failed int)) (int, int, error) {
	totalProcessed := 0
	totalFailed := 0

	for i := 0; i < len(items); i += p.batchSize {
		end := i + p.batchSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		err := p.pool.SubmitBlocking(context.Background(), func(ctx context.Context) error {
			processed, failed, err := p.processChunk(clientID, chunk, uploadID)
			totalProcessed += processed
			totalFailed += failed
			onProgress(processed, failed)
			return err
		})

		if err != nil {
			p.log.Warn().Str("reason", "pool_full").Int("chunk", i/p.batchSize).Err(err).Msg("batch_submission_deferred")
		}
	}

	return totalProcessed, totalFailed, nil
}

func (p *Processor) processChunk(clientID uuid.UUID, items []inventory.BulkItemInput, uploadID uuid.UUID) (int, int, error) {
	log := p.log.WithSessionID(uploadID.String())
	ctx := context.Background()

	var processed, failed int
	var lastErr error

	for _, item := range items {
		itemCopy := item
		hash := computeItemHash(item)
		itemCopy.LastModifiedHash = &hash

		if err := p.processItem(ctx, clientID, itemCopy, uploadID); err != nil {
			failed++
			lastErr = err
			log.Warn().Str("sku", item.SKU).Err(err).Msg("item_processing_failed")
		} else {
			processed++
		}

		if err := p.uploadRepo.UpdateProgress(ctx, uploadID, processed, failed); err != nil {
			log.Warn().Err(err).Msg("progress_update_failed")
		}
	}

	if failed > 0 && processed == 0 {
		return processed, failed, lastErr
	}
	return processed, failed, nil
}

func (p *Processor) processItem(ctx context.Context, clientID uuid.UUID, item inventory.BulkItemInput, uploadID uuid.UUID) error {
	tx, err := p.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	product := &inventory.Product{
		SKU:              item.SKU,
		ClientID:         clientID,
		Name:             item.Name,
		Description:      item.Description,
		Category:         item.Category,
		Manufacturer:     item.Manufacturer,
		LastModifiedHash: item.LastModifiedHash,
	}
	if err := p.inventoryRepo.UpsertProduct(ctx, tx, product); err != nil {
		return err
	}

	inv := &inventory.Inventory{
		Quantity:          item.Quantity,
		ReservedQuantity:  item.ReservedQuantity,
		ReorderPoint:      item.ReorderPoint,
		WarehouseLocation: item.WarehouseLocation,
		Status:            inventory.InventoryStatusActive,
	}
	if err := p.inventoryRepo.UpsertInventory(ctx, tx, product.ID, inv); err != nil {
		return err
	}

	pr := &inventory.Pricing{
		UnitPrice:        item.UnitPrice,
		Currency:         item.Currency,
		MinOrderQuantity: 1,
		BulkPrice:        item.BulkPrice,
	}
	if err := p.inventoryRepo.UpsertPricing(ctx, tx, product.ID, pr); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if p.cache != nil {
		cacheKey := "product:" + clientID.String() + ":" + item.SKU
		p.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func computeItemHash(item inventory.BulkItemInput) string {
	data, _ := json.Marshal(item)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (p *Processor) Start() {
	p.pool.Start()
}

func (p *Processor) Shutdown() {
	p.pool.Shutdown()
}