package domain

import (
	"time"

	"github.com/google/uuid"
)

type UploadStatus string

const (
	UploadStatusPending       UploadStatus = "pending"
	UploadStatusInProgress    UploadStatus = "in_progress"
	UploadStatusCompleted     UploadStatus = "completed"
	UploadStatusFailed        UploadStatus = "failed"
	UploadStatusPartialFailure UploadStatus = "partial_failure"
)

type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "pending"
	BatchStatusProcessing BatchStatus = "processing"
	BatchStatusCompleted  BatchStatus = "completed"
	BatchStatusFailed     BatchStatus = "failed"
)

type InventoryUpload struct {
	ID             uuid.UUID
	SessionID      uuid.UUID
	ClientID       uuid.UUID
	IdempotencyKey string
	TotalItems     int
	ProcessedItems int
	FailedItems    int
	Status         UploadStatus
	ErrorSummary   *string
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

type UploadBatch struct {
	ID            uuid.UUID
	UploadID      uuid.UUID
	BatchNumber   int
	ItemsCount    int
	ProcessedCount int
	FailedCount   int
	Status        BatchStatus
	ErrorDetail   *string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

func (u *InventoryUpload) Progress() float64 {
	if u.TotalItems == 0 {
		return 0
	}
	return float64(u.ProcessedItems+u.FailedItems) / float64(u.TotalItems) * 100
}

func (u *InventoryUpload) IsTerminal() bool {
	return u.Status == UploadStatusCompleted ||
		u.Status == UploadStatusFailed ||
		u.Status == UploadStatusPartialFailure
}