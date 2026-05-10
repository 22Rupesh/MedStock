package domain

import (
	"time"

	"github.com/google/uuid"
)

type InventoryStatus string

const (
	InventoryStatusActive       InventoryStatus = "active"
	InventoryStatusInactive    InventoryStatus = "inactive"
	InventoryStatusDiscontinued InventoryStatus = "discontinued"
)

type Product struct {
	ID               uuid.UUID
	SKU              string
	ClientID         uuid.UUID
	Name             string
	Description      *string
	Category         *string
	Manufacturer     *string
	Version          int
	LastModifiedHash *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Inventory        *Inventory
	Pricing          *Pricing
}

type Inventory struct {
	ID                 uuid.UUID
	ProductID          uuid.UUID
	Quantity           int
	ReservedQuantity   int
	AvailableQuantity  int
	ReorderPoint       *int
	WarehouseLocation  *string
	LastStockCheck     *time.Time
	Status             InventoryStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Pricing struct {
	ID                uuid.UUID
	ProductID         uuid.UUID
	UnitPrice         float64
	Currency          string
	MinOrderQuantity  int
	BulkPrice         *float64
	EffectiveFrom     time.Time
	EffectiveUntil    *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StockEvent struct {
	ID             uuid.UUID
	ProductID      uuid.UUID
	UploadID       *uuid.UUID
	EventType      string
	QuantityChange int
	QuantityBefore int
	QuantityAfter  int
	Reason         *string
	CreatedAt      time.Time
}

type BulkItemInput struct {
	SKU               string   `json:"sku" validate:"required,max=100"`
	Name              string   `json:"name" validate:"required,max=255"`
	Description       *string  `json:"description"`
	Category          *string  `json:"category" validate:"omitempty,max=100"`
	Manufacturer      *string  `json:"manufacturer" validate:"omitempty,max=255"`
	Quantity          int      `json:"quantity" validate:"gte=0"`
	ReservedQuantity  int      `json:"reserved_quantity" validate:"gte=0"`
	ReorderPoint      *int     `json:"reorder_point"`
	WarehouseLocation *string  `json:"warehouse_location" validate:"omitempty,max=100"`
	UnitPrice         float64  `json:"unit_price" validate:"required,gte=0"`
	Currency          string   `json:"currency" validate:"required,len=3"`
	BulkPrice         *float64 `json:"bulk_price" validate:"omitempty,gte=0"`
	LastModifiedHash  *string  `json:"-"`
}

type BulkUploadRequest struct {
	SessionID      uuid.UUID      `json:"session_id"`
	IdempotencyKey string         `json:"idempotency_key" validate:"required,max=128"`
	Items          []BulkItemInput `json:"items" validate:"required,min=1,max=10000,dive"`
}

type BulkUploadResponse struct {
	SessionID    uuid.UUID `json:"session_id"`
	Status       string    `json:"status"`
	TotalItems   int       `json:"total_items"`
	Processed    int       `json:"processed"`
	Failed       int       `json:"failed"`
	Progress     float64   `json:"progress"`
	DurationMs   int64     `json:"duration_ms"`
	IsIdempotent bool      `json:"is_idempotent"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type ProductListResponse struct {
	Products   []Product  `json:"products"`
	Pagination Pagination `json:"pagination"`
}