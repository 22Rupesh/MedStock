package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	uploadDomain "medstock/internal/domain/upload"
)

type UploadRepository struct {
	pool *pgxpool.Pool
}

func NewUploadRepository(pool *pgxpool.Pool) *UploadRepository {
	return &UploadRepository{pool: pool}
}

func (r *UploadRepository) Create(ctx context.Context, u *uploadDomain.InventoryUpload) error {
	query := `
		INSERT INTO inventory_uploads (id, session_id, client_id, idempotency_key, total_items, processed_items, failed_items, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.SessionID, u.ClientID, u.IdempotencyKey,
		u.TotalItems, u.ProcessedItems, u.FailedItems,
		u.Status, u.CreatedAt,
	)
	return err
}

func (r *UploadRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*uploadDomain.InventoryUpload, error) {
	query := `
		SELECT id, session_id, client_id, idempotency_key, total_items, processed_items, failed_items,
		       status, error_summary, created_at, started_at, completed_at
		FROM inventory_uploads
		WHERE session_id = $1
	`

	var u uploadDomain.InventoryUpload
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&u.ID, &u.SessionID, &u.ClientID, &u.IdempotencyKey,
		&u.TotalItems, &u.ProcessedItems, &u.FailedItems,
		&u.Status, &u.ErrorSummary, &u.CreatedAt, &u.StartedAt, &u.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UploadRepository) GetByIdempotencyKey(ctx context.Context, clientID uuid.UUID, key string) (*uploadDomain.InventoryUpload, error) {
	query := `
		SELECT id, session_id, client_id, idempotency_key, total_items, processed_items, failed_items,
		       status, error_summary, created_at, started_at, completed_at
		FROM inventory_uploads
		WHERE client_id = $1 AND idempotency_key = $2
	`

	var u uploadDomain.InventoryUpload
	err := r.pool.QueryRow(ctx, query, clientID, key).Scan(
		&u.ID, &u.SessionID, &u.ClientID, &u.IdempotencyKey,
		&u.TotalItems, &u.ProcessedItems, &u.FailedItems,
		&u.Status, &u.ErrorSummary, &u.CreatedAt, &u.StartedAt, &u.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UploadRepository) UpdateProgress(ctx context.Context, id uuid.UUID, processed, failed int) error {
	query := `
		UPDATE inventory_uploads
		SET processed_items = $2, failed_items = $3, started_at = COALESCE(started_at, NOW())
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, processed, failed)
	return err
}

func (r *UploadRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status uploadDomain.UploadStatus, errSummary *string) error {
	var query string
	var args []any

	if status == uploadDomain.UploadStatusCompleted || status == uploadDomain.UploadStatusFailed || status == uploadDomain.UploadStatusPartialFailure {
		query = `
			UPDATE inventory_uploads
			SET status = $2, error_summary = $3, completed_at = $4
			WHERE id = $1
		`
		now := time.Now()
		args = []any{id, status, errSummary, now}
	} else {
		query = `UPDATE inventory_uploads SET status = $2, error_summary = $3 WHERE id = $1`
		args = []any{id, status, errSummary}
	}

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *UploadRepository) CreateBatch(ctx context.Context, b *uploadDomain.UploadBatch) error {
	query := `
		INSERT INTO upload_batches (id, upload_id, batch_number, items_count, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, b.ID, b.UploadID, b.BatchNumber, b.ItemsCount, b.Status, b.CreatedAt)
	return err
}

func (r *UploadRepository) UpdateBatchProgress(ctx context.Context, id uuid.UUID, processed, failed int) error {
	query := `
		UPDATE upload_batches
		SET processed_count = $2, failed_count = $3
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, processed, failed)
	return err
}

func (r *UploadRepository) UpdateBatchStatus(ctx context.Context, id uuid.UUID, status uploadDomain.BatchStatus, errDetail *string) error {
	var query string
	var args []any

	if status == uploadDomain.BatchStatusCompleted || status == uploadDomain.BatchStatusFailed {
		query = `
			UPDATE upload_batches
			SET status = $2, error_detail = $3, completed_at = $4
			WHERE id = $1
		`
		now := time.Now()
		args = []any{id, status, errDetail, now}
	} else {
		query = `UPDATE upload_batches SET status = $2, error_detail = $3 WHERE id = $1`
		args = []any{id, status, errDetail}
	}

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *UploadRepository) GetBatchesByUploadID(ctx context.Context, uploadID uuid.UUID) ([]uploadDomain.UploadBatch, error) {
	query := `
		SELECT id, upload_id, batch_number, items_count, processed_count, failed_count,
		       status, error_detail, created_at, completed_at
		FROM upload_batches
		WHERE upload_id = $1
		ORDER BY batch_number
	`

	rows, err := r.pool.Query(ctx, query, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []uploadDomain.UploadBatch
	for rows.Next() {
		var b uploadDomain.UploadBatch
		if err := rows.Scan(&b.ID, &b.UploadID, &b.BatchNumber, &b.ItemsCount,
			&b.ProcessedCount, &b.FailedCount, &b.Status, &b.ErrorDetail,
			&b.CreatedAt, &b.CompletedAt); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}