-- +goose Up
CREATE TABLE material_stocks (
    material_id UUID PRIMARY KEY REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_material_stock_qty_non_negative CHECK (qty >= 0)
);

CREATE TYPE stock_movement_type AS ENUM (
    'IN',
    'OUT',
    'ADJUSTMENT_IN',
    'ADJUSTMENT_OUT'
);

CREATE TYPE stock_reference_type AS ENUM (
    'PO_RECEIPT',
    'SHORTAGE_PURCHASE',
    'ADDITIONAL_REQUIREMENT',
    'MATERIAL_USAGE',
    'STOCK_ADJUSTMENT'
);

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    movement_type stock_movement_type NOT NULL,
    reference_type stock_reference_type NOT NULL,
    reference_id UUID NOT NULL,
    qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    movement_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_movement_qty_positive CHECK (qty > 0)
);

CREATE INDEX idx_stock_movements_material_date
    ON stock_movements(material_id, movement_date DESC);
CREATE INDEX idx_stock_movements_reference
    ON stock_movements(reference_type, reference_id);

CREATE TYPE stock_reservation_status AS ENUM (
    'ACTIVE',
    'CONSUMED',
    'RELEASED'
);

CREATE TABLE stock_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    procurement_request_item_id UUID,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    status stock_reservation_status NOT NULL DEFAULT 'ACTIVE',
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_reservation_qty_positive CHECK (qty > 0),
    CONSTRAINT chk_stock_reservation_lifecycle CHECK (
        (status = 'ACTIVE' AND released_at IS NULL AND consumed_at IS NULL)
        OR (status = 'RELEASED' AND released_at IS NOT NULL AND consumed_at IS NULL)
        OR (status = 'CONSUMED' AND consumed_at IS NOT NULL AND released_at IS NULL)
    )
);

CREATE INDEX idx_stock_reservations_material_status
    ON stock_reservations(material_id, status);
CREATE INDEX idx_stock_reservations_scheduled_menu
    ON stock_reservations(scheduled_menu_id);
CREATE INDEX idx_stock_reservations_procurement_item
    ON stock_reservations(procurement_request_item_id)
    WHERE procurement_request_item_id IS NOT NULL;

-- `procurement_request_item_id` intentionally has no foreign key yet.
-- Migration 000004_procurement will add the FK after procurement_request_items exists.

-- +goose Down
DROP TABLE IF EXISTS stock_reservations;
DROP TYPE IF EXISTS stock_reservation_status;
DROP TABLE IF EXISTS stock_movements;
DROP TYPE IF EXISTS stock_reference_type;
DROP TYPE IF EXISTS stock_movement_type;
DROP TABLE IF EXISTS material_stocks;
