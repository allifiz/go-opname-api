-- +goose Up
CREATE TYPE procurement_request_status AS ENUM (
    'DRAFT',
    'WAITING_VERIFICATION',
    'VERIFIED',
    'REJECTED'
);

CREATE TABLE procurement_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    status procurement_request_status NOT NULL DEFAULT 'DRAFT',
    checked_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    checked_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_procurement_request_verification CHECK (
        (status <> 'VERIFIED')
        OR (verified_by IS NOT NULL AND verified_at IS NOT NULL)
    )
);

CREATE INDEX idx_procurement_requests_scheduled_menu
    ON procurement_requests(scheduled_menu_id);
CREATE INDEX idx_procurement_requests_status
    ON procurement_requests(status);

CREATE TABLE procurement_request_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    procurement_request_id UUID NOT NULL REFERENCES procurement_requests(id) ON UPDATE CASCADE ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    gross_requirement_qty NUMERIC(18,4) NOT NULL,
    existing_stock_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    reserved_stock_qty NUMERIC(18,4) NOT NULL DEFAULT 0,
    net_procurement_qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_procurement_gross_requirement_non_negative CHECK (gross_requirement_qty >= 0),
    CONSTRAINT chk_procurement_existing_stock_non_negative CHECK (existing_stock_qty >= 0),
    CONSTRAINT chk_procurement_reserved_stock_non_negative CHECK (reserved_stock_qty >= 0),
    CONSTRAINT chk_procurement_net_qty_non_negative CHECK (net_procurement_qty >= 0),
    CONSTRAINT chk_procurement_net_not_above_gross CHECK (net_procurement_qty <= gross_requirement_qty),
    CONSTRAINT uq_procurement_request_material UNIQUE (procurement_request_id, material_id)
);

CREATE INDEX idx_procurement_request_items_material
    ON procurement_request_items(material_id);

ALTER TABLE stock_reservations
    ADD CONSTRAINT fk_stock_reservations_procurement_request_item
    FOREIGN KEY (procurement_request_item_id)
    REFERENCES procurement_request_items(id)
    ON UPDATE CASCADE
    ON DELETE RESTRICT;

CREATE TYPE purchase_order_status AS ENUM (
    'DRAFT',
    'VERIFIED',
    'PARTIALLY_RECEIVED',
    'COMPLETED'
);

CREATE TABLE purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    procurement_request_id UUID NOT NULL REFERENCES procurement_requests(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    po_number VARCHAR(100) NOT NULL UNIQUE,
    delivery_date DATE NOT NULL,
    status purchase_order_status NOT NULL DEFAULT 'DRAFT',
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_orders_procurement_request
    ON purchase_orders(procurement_request_id);
CREATE INDEX idx_purchase_orders_scheduled_menu
    ON purchase_orders(scheduled_menu_id);
CREATE INDEX idx_purchase_orders_delivery_date
    ON purchase_orders(delivery_date);

CREATE TYPE purchase_order_item_status AS ENUM (
    'WAITING',
    'CANCELLED',
    'NOT_RECEIVED',
    'PARTIAL_RECEIVED',
    'RECEIVED',
    'OVER_RECEIVED',
    'FULFILLED'
);

CREATE TABLE purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON UPDATE CASCADE ON DELETE CASCADE,
    procurement_request_item_id UUID NOT NULL REFERENCES procurement_request_items(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    ordered_qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    agreed_unit_price NUMERIC(18,2) NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    status purchase_order_item_status NOT NULL DEFAULT 'WAITING',
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    cancel_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_purchase_order_item_qty_positive CHECK (ordered_qty > 0),
    CONSTRAINT chk_purchase_order_item_price_non_negative CHECK (agreed_unit_price >= 0),
    CONSTRAINT chk_purchase_order_item_cancel_lifecycle CHECK (
        (status = 'CANCELLED' AND cancelled_at IS NOT NULL AND cancelled_by IS NOT NULL AND cancel_reason IS NOT NULL)
        OR (status <> 'CANCELLED' AND cancelled_at IS NULL AND cancelled_by IS NULL AND cancel_reason IS NULL)
    ),
    CONSTRAINT uq_purchase_order_material UNIQUE (purchase_order_id, material_id)
);

CREATE INDEX idx_purchase_order_items_procurement_item
    ON purchase_order_items(procurement_request_item_id);
CREATE INDEX idx_purchase_order_items_status
    ON purchase_order_items(status);

-- H-1 cancellation timing depends on the purchase order delivery date and is
-- enforced transactionally by the service layer before changing an item to CANCELLED.

-- +goose Down
DROP TABLE IF EXISTS purchase_order_items;
DROP TYPE IF EXISTS purchase_order_item_status;
DROP TABLE IF EXISTS purchase_orders;
DROP TYPE IF EXISTS purchase_order_status;
ALTER TABLE stock_reservations
    DROP CONSTRAINT IF EXISTS fk_stock_reservations_procurement_request_item;
DROP TABLE IF EXISTS procurement_request_items;
DROP TABLE IF EXISTS procurement_requests;
DROP TYPE IF EXISTS procurement_request_status;
