-- +goose Up
CREATE TABLE receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_receipts_purchase_order
    ON receipts(purchase_order_id, received_at DESC);

CREATE TABLE receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON UPDATE CASCADE ON DELETE CASCADE,
    purchase_order_item_id UUID NOT NULL REFERENCES purchase_order_items(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    received_qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    agreed_unit_price NUMERIC(18,2) NOT NULL,
    actual_amount NUMERIC(18,2) GENERATED ALWAYS AS (ROUND(received_qty * agreed_unit_price, 2)) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_receipt_item_qty_positive CHECK (received_qty > 0),
    CONSTRAINT chk_receipt_item_price_non_negative CHECK (agreed_unit_price >= 0),
    CONSTRAINT uq_receipt_purchase_order_item UNIQUE (receipt_id, purchase_order_item_id)
);

CREATE INDEX idx_receipt_items_purchase_order_item
    ON receipt_items(purchase_order_item_id);
CREATE INDEX idx_receipt_items_material
    ON receipt_items(material_id);

CREATE TYPE receipt_document_type AS ENUM (
    'INVOICE',
    'NOTA',
    'PHOTO',
    'OTHER'
);

CREATE TABLE receipt_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts(id) ON UPDATE CASCADE ON DELETE CASCADE,
    document_type receipt_document_type NOT NULL,
    file_url TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    uploaded_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_receipt_document_file_url_not_blank CHECK (BTRIM(file_url) <> ''),
    CONSTRAINT chk_receipt_document_file_name_not_blank CHECK (BTRIM(file_name) <> '')
);

CREATE INDEX idx_receipt_documents_receipt
    ON receipt_documents(receipt_id, created_at ASC);

-- +goose Down
DROP TABLE IF EXISTS receipt_documents;
DROP TYPE IF EXISTS receipt_document_type;
DROP TABLE IF EXISTS receipt_items;
DROP TABLE IF EXISTS receipts;
