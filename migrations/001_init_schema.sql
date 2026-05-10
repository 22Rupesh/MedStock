-- Migration: 001_init_schema
-- Description: Initial schema for MedStock inventory system

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE upload_status AS ENUM ('pending', 'in_progress', 'completed', 'failed', 'partial_failure');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE inventory_status AS ENUM ('active', 'inactive', 'discontinued');

CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(64) UNIQUE NOT NULL,
    rate_limit INT DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_api_key ON clients(api_key);

CREATE TABLE inventory_uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL UNIQUE,
    client_id UUID NOT NULL REFERENCES clients(id),
    idempotency_key VARCHAR(128) NOT NULL,
    total_items INT NOT NULL,
    processed_items INT NOT NULL DEFAULT 0,
    failed_items INT NOT NULL DEFAULT 0,
    status upload_status NOT NULL DEFAULT 'pending',
    error_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    CONSTRAINT unique_client_idempotency UNIQUE (client_id, idempotency_key)
);

CREATE INDEX idx_uploads_client_id ON inventory_uploads(client_id);
CREATE INDEX idx_uploads_status ON inventory_uploads(status);
CREATE INDEX idx_uploads_created_at ON inventory_uploads(created_at);

CREATE TABLE upload_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    upload_id UUID NOT NULL REFERENCES inventory_uploads(id) ON DELETE CASCADE,
    batch_number INT NOT NULL,
    items_count INT NOT NULL,
    processed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    status batch_status NOT NULL DEFAULT 'pending',
    error_detail TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT unique_upload_batch UNIQUE (upload_id, batch_number)
);

CREATE INDEX idx_batches_upload_id ON upload_batches(upload_id);
CREATE INDEX idx_batches_status ON upload_batches(status);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sku VARCHAR(100) NOT NULL,
    client_id UUID NOT NULL REFERENCES clients(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    manufacturer VARCHAR(255),
    version INT NOT NULL DEFAULT 1,
    last_modified_hash VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_client_sku UNIQUE (client_id, sku)
);

CREATE INDEX idx_products_client_id ON products(client_id);
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_updated_at ON products(updated_at);

CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 0,
    reserved_quantity INT NOT NULL DEFAULT 0,
    available_quantity INT GENERATED ALWAYS AS (quantity - reserved_quantity) STORED,
    reorder_point INT,
    warehouse_location VARCHAR(100),
    last_stock_check TIMESTAMPTZ,
    status inventory_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT positive_quantity CHECK (quantity >= 0),
    CONSTRAINT positive_reserved CHECK (reserved_quantity >= 0)
);

CREATE INDEX idx_inventory_product_id ON inventory(product_id);
CREATE INDEX idx_inventory_status ON inventory(status);
CREATE INDEX idx_inventory_warehouse ON inventory(warehouse_location);

CREATE TABLE pricing (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    unit_price DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    min_order_quantity INT DEFAULT 1,
    bulk_price DECIMAL(12, 2),
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pricing_product_id ON pricing(product_id);
CREATE INDEX idx_pricing_effective ON pricing(effective_from, effective_until);

CREATE TABLE stock_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id),
    upload_id UUID REFERENCES inventory_uploads(id),
    event_type VARCHAR(50) NOT NULL,
    quantity_change INT NOT NULL,
    quantity_before INT NOT NULL,
    quantity_after INT NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_product_id ON stock_events(product_id);
CREATE INDEX idx_events_upload_id ON stock_events(upload_id);
CREATE INDEX idx_events_created_at ON stock_events(created_at);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_clients_updated_at
    BEFORE UPDATE ON clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_inventory_updated_at
    BEFORE UPDATE ON inventory
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_updated_at
    BEFORE UPDATE ON pricing
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();