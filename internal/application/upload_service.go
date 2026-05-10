package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	clientdomain "medstock/internal/domain"
	inventory "medstock/internal/domain/inventory"
	upload "medstock/internal/domain/upload"
	"medstock/internal/infrastructure/redis"
	"medstock/internal/worker"
	"medstock/pkg/logger"
)

var (
	ErrUploadNotFound   = errors.New("upload not found")
	ErrDuplicateUpload  = errors.New("duplicate upload detected")
	ErrInvalidItemCount = errors.New("item count exceeds maximum")
	ErrPoolExhausted    = errors.New("worker pool exhausted")
)

type BulkItemInput = inventory.BulkItemInput

type UploadService struct {
	uploadRepo     UploadRepo
	inventoryRepo  InventoryRepo
	clientRepo     ClientRepo
	processor      *worker.Processor
	cache          Cache
	log            *logger.Logger
	maxItemsPerBatch int
}

type UploadRepo interface {
	Create(ctx context.Context, u *upload.InventoryUpload) error
	GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*upload.InventoryUpload, error)
	GetByIdempotencyKey(ctx context.Context, clientID uuid.UUID, key string) (*upload.InventoryUpload, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, processed, failed int) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status upload.UploadStatus, errSummary *string) error
	CreateBatch(ctx context.Context, b *upload.UploadBatch) error
	UpdateBatchProgress(ctx context.Context, id uuid.UUID, processed, failed int) error
	UpdateBatchStatus(ctx context.Context, id uuid.UUID, status upload.BatchStatus, errDetail *string) error
	GetBatchesByUploadID(ctx context.Context, uploadID uuid.UUID) ([]upload.UploadBatch, error)
}

type InventoryRepo interface {
	UpsertProduct(ctx context.Context, tx pgx.Tx, p *inventory.Product) error
	UpsertInventory(ctx context.Context, tx pgx.Tx, productID uuid.UUID, inv *inventory.Inventory) error
	UpsertPricing(ctx context.Context, tx pgx.Tx, productID uuid.UUID, pr *inventory.Pricing) error
	RecordStockEvent(ctx context.Context, tx pgx.Tx, event *inventory.StockEvent) error
	ListProducts(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]inventory.Product, int, error)
	GetProductBySKU(ctx context.Context, clientID uuid.UUID, sku string) (*inventory.Product, error)
}

type ClientRepo interface {
	GetByAPIKey(ctx context.Context, apiKey string) (*clientdomain.Client, error)
}

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
}

func NewUploadService(cfg UploadServiceConfig) *UploadService {
	return &UploadService{
		uploadRepo:     cfg.UploadRepo,
		inventoryRepo:  cfg.InventoryRepo,
		clientRepo:     cfg.ClientRepo,
		processor:      cfg.Processor,
		cache:          cfg.Cache,
		log:            cfg.Logger,
		maxItemsPerBatch: 10000,
	}
}

type UploadServiceConfig struct {
	UploadRepo    UploadRepo
	InventoryRepo InventoryRepo
	ClientRepo    ClientRepo
	Processor     *worker.Processor
	Cache         Cache
	Logger        *logger.Logger
}

func (s *UploadService) ProcessBulkUpload(ctx context.Context, clientID uuid.UUID, req *inventory.BulkUploadRequest) (*inventory.BulkUploadResponse, error) {
	start := time.Now()

	if len(req.Items) > s.maxItemsPerBatch {
		return nil, ErrInvalidItemCount
	}

	existing, err := s.uploadRepo.GetByIdempotencyKey(ctx, clientID, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}
	if existing != nil {
		if existing.Status == upload.UploadStatusCompleted {
			return &inventory.BulkUploadResponse{
				SessionID:    existing.SessionID,
				Status:       string(existing.Status),
				TotalItems:   existing.TotalItems,
				Processed:    existing.ProcessedItems,
				Failed:       existing.FailedItems,
				Progress:     existing.Progress(),
				DurationMs:   time.Since(start).Milliseconds(),
				IsIdempotent: true,
			}, nil
		}
		return nil, ErrDuplicateUpload
	}

	uploadRecord := &upload.InventoryUpload{
		ID:             uuid.New(),
		SessionID:      req.SessionID,
		ClientID:       clientID,
		IdempotencyKey: req.IdempotencyKey,
		TotalItems:     len(req.Items),
		Status:         upload.UploadStatusPending,
		CreatedAt:      time.Now(),
	}

	if err := s.uploadRepo.Create(ctx, uploadRecord); err != nil {
		return nil, fmt.Errorf("create upload record: %w", err)
	}

	s.uploadRepo.UpdateStatus(ctx, uploadRecord.ID, upload.UploadStatusInProgress, nil)

	s.log.Info().
		Str("session_id", req.SessionID.String()).
		Int("item_count", len(req.Items)).
		Msg("bulk_upload_started")

	processed, failed, err := s.processor.ProcessChunks(
		clientID,
		req.Items,
		uploadRecord.ID,
		func(proc, fail int) {
			s.uploadRepo.UpdateProgress(ctx, uploadRecord.ID, proc, fail)
		},
	)

	var finalStatus upload.UploadStatus
	var errSummary *string
	if err != nil && failed == 0 {
		finalStatus = upload.UploadStatusFailed
		errStr := err.Error()
		errSummary = &errStr
	} else if failed > 0 && processed == 0 {
		finalStatus = upload.UploadStatusFailed
		errStr := "all items failed"
		errSummary = &errStr
	} else if failed > 0 {
		finalStatus = upload.UploadStatusPartialFailure
		summary := fmt.Sprintf("%d processed, %d failed", processed, failed)
		errSummary = &summary
	} else {
		finalStatus = upload.UploadStatusCompleted
	}

	s.uploadRepo.UpdateStatus(ctx, uploadRecord.ID, finalStatus, errSummary)

	if s.cache != nil {
		s.cache.Delete(ctx, redis.CatalogKey(clientID.String()))
	}

	s.log.Info().
		Str("session_id", req.SessionID.String()).
		Int("processed", processed).
		Int("failed", failed).
		Str("status", string(finalStatus)).
		Int64("duration_ms", time.Since(start).Milliseconds()).
		Msg("bulk_upload_completed")

	return &inventory.BulkUploadResponse{
		SessionID:    req.SessionID,
		Status:       string(finalStatus),
		TotalItems:   len(req.Items),
		Processed:    processed,
		Failed:       failed,
		Progress:     100,
		DurationMs:   time.Since(start).Milliseconds(),
		IsIdempotent: false,
	}, nil
}

func (s *UploadService) GetUploadStatus(ctx context.Context, sessionID uuid.UUID) (*upload.InventoryUpload, error) {
	upload, err := s.uploadRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if upload == nil {
		return nil, ErrUploadNotFound
	}
	return upload, nil
}