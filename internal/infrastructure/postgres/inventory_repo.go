package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	inventory "medstock/internal/domain/inventory"
)

type InventoryRepository struct {
	pool *pgxpool.Pool
}

func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

func (r *InventoryRepository) UpsertProduct(ctx context.Context, tx pgx.Tx, p *inventory.Product) error {
	query := `
		INSERT INTO products (id, sku, client_id, name, description, category, manufacturer, version, last_modified_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (client_id, sku) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			manufacturer = EXCLUDED.manufacturer,
			version = products.version + 1,
			last_modified_hash = EXCLUDED.last_modified_hash,
			updated_at = NOW()
		RETURNING id, version, created_at, updated_at
	`
	var newVersion int
	err := tx.QueryRow(ctx, query,
		uuid.New(), p.SKU, p.ClientID, p.Name, p.Description,
		p.Category, p.Manufacturer, 1, p.LastModifiedHash, time.Now(), time.Now(),
	).Scan(&p.ID, &newVersion, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return err
	}
	p.Version = newVersion
	return nil
}

func (r *InventoryRepository) GetProductBySKU(ctx context.Context, clientID uuid.UUID, sku string) (*inventory.Product, error) {
	query := `
		SELECT id, sku, client_id, name, description, category, manufacturer, version,
		       last_modified_hash, created_at, updated_at
		FROM products
		WHERE client_id = $1 AND sku = $2
	`

	var p inventory.Product
	err := r.pool.QueryRow(ctx, query, clientID, sku).Scan(
		&p.ID, &p.SKU, &p.ClientID, &p.Name, &p.Description,
		&p.Category, &p.Manufacturer, &p.Version,
		&p.LastModifiedHash, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *InventoryRepository) UpsertInventory(ctx context.Context, tx pgx.Tx, productID uuid.UUID, inv *inventory.Inventory) error {
	query := `
		INSERT INTO inventory (id, product_id, quantity, reserved_quantity, reorder_point, warehouse_location, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (product_id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			reserved_quantity = EXCLUDED.reserved_quantity,
			reorder_point = EXCLUDED.reorder_point,
			warehouse_location = EXCLUDED.warehouse_location,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, query,
		uuid.New(), productID, inv.Quantity, inv.ReservedQuantity,
		inv.ReorderPoint, inv.WarehouseLocation, inv.Status, time.Now(), time.Now(),
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return err
	}
	inv.ProductID = productID
	return nil
}

func (r *InventoryRepository) UpsertPricing(ctx context.Context, tx pgx.Tx, productID uuid.UUID, pr *inventory.Pricing) error {
	query := `
		INSERT INTO pricing (id, product_id, unit_price, currency, min_order_quantity, bulk_price, effective_from, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (product_id) DO UPDATE SET
			unit_price = EXCLUDED.unit_price,
			currency = EXCLUDED.currency,
			min_order_quantity = EXCLUDED.min_order_quantity,
			bulk_price = EXCLUDED.bulk_price,
			effective_until = NOW(),
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, query,
		uuid.New(), productID, pr.UnitPrice, pr.Currency,
		pr.MinOrderQuantity, pr.BulkPrice, time.Now(), time.Now(), time.Now(),
	).Scan(&pr.ID, &pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return err
	}
	pr.ProductID = productID
	return nil
}

func (r *InventoryRepository) RecordStockEvent(ctx context.Context, tx pgx.Tx, event *inventory.StockEvent) error {
	query := `
		INSERT INTO stock_events (id, product_id, upload_id, event_type, quantity_change, quantity_before, quantity_after, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.Exec(ctx, query,
		event.ID, event.ProductID, event.UploadID, event.EventType,
		event.QuantityChange, event.QuantityBefore, event.QuantityAfter,
		event.Reason, time.Now(),
	)
	return err
}

func (r *InventoryRepository) GetInventoryByProductID(ctx context.Context, productID uuid.UUID) (*inventory.Inventory, error) {
	query := `
		SELECT id, product_id, quantity, reserved_quantity, reorder_point, warehouse_location,
		       status, created_at, updated_at
		FROM inventory
		WHERE product_id = $1
	`

	var inv inventory.Inventory
	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&inv.ID, &inv.ProductID, &inv.Quantity, &inv.ReservedQuantity,
		&inv.ReorderPoint, &inv.WarehouseLocation, &inv.Status,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *InventoryRepository) ListProducts(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]inventory.Product, int, error) {
	countQuery := `SELECT COUNT(*) FROM products WHERE client_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, clientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, sku, client_id, name, description, category, manufacturer, version,
		       last_modified_hash, created_at, updated_at
		FROM products
		WHERE client_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, clientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []inventory.Product
	for rows.Next() {
		var p inventory.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.ClientID, &p.Name, &p.Description,
			&p.Category, &p.Manufacturer, &p.Version,
			&p.LastModifiedHash, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}