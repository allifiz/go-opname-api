-- +goose Up
CREATE TYPE direct_purchase_type AS ENUM (
    'SHORTAGE',
    'ADDITIONAL_REQUIREMENT'
);

CREATE TABLE additional_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    previous_portions INTEGER NOT NULL,
    new_portions INTEGER NOT NULL,
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_additional_requirement_portions CHECK (
        previous_portions > 0 AND new_portions > previous_portions
    )
);

CREATE INDEX idx_additional_requirements_scheduled_menu
    ON additional_requirements(scheduled_menu_id, created_at DESC);

CREATE TABLE additional_requirement_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    additional_requirement_id UUID NOT NULL REFERENCES additional_requirements(id) ON UPDATE CASCADE ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    additional_qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_additional_requirement_qty_positive CHECK (additional_qty > 0),
    CONSTRAINT uq_additional_requirement_material UNIQUE (additional_requirement_id, material_id)
);

CREATE INDEX idx_additional_requirement_items_material
    ON additional_requirement_items(material_id);

CREATE TABLE direct_purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    purchase_type direct_purchase_type NOT NULL,
    source_name VARCHAR(200) NOT NULL,
    purchase_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    purchased_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_direct_purchase_source_not_blank CHECK (BTRIM(source_name) <> '')
);

CREATE INDEX idx_direct_purchases_scheduled_menu
    ON direct_purchases(scheduled_menu_id, purchase_date DESC);
CREATE INDEX idx_direct_purchases_type
    ON direct_purchases(purchase_type);

CREATE TABLE direct_purchase_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    direct_purchase_id UUID NOT NULL REFERENCES direct_purchases(id) ON UPDATE CASCADE ON DELETE CASCADE,
    purchase_order_item_id UUID REFERENCES purchase_order_items(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    additional_requirement_item_id UUID REFERENCES additional_requirement_items(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    unit_price NUMERIC(18,2) NOT NULL,
    total_amount NUMERIC(18,2) GENERATED ALWAYS AS (ROUND(qty * unit_price, 2)) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_direct_purchase_item_qty_positive CHECK (qty > 0),
    CONSTRAINT chk_direct_purchase_item_price_non_negative CHECK (unit_price >= 0),
    CONSTRAINT chk_direct_purchase_item_single_source CHECK (
        (purchase_order_item_id IS NOT NULL AND additional_requirement_item_id IS NULL)
        OR (purchase_order_item_id IS NULL AND additional_requirement_item_id IS NOT NULL)
    )
);

CREATE INDEX idx_direct_purchase_items_po_item
    ON direct_purchase_items(purchase_order_item_id)
    WHERE purchase_order_item_id IS NOT NULL;
CREATE INDEX idx_direct_purchase_items_additional_item
    ON direct_purchase_items(additional_requirement_item_id)
    WHERE additional_requirement_item_id IS NOT NULL;
CREATE INDEX idx_direct_purchase_items_material
    ON direct_purchase_items(material_id);

-- +goose Down
DROP TABLE IF EXISTS direct_purchase_items;
DROP TABLE IF EXISTS direct_purchases;
DROP TABLE IF EXISTS additional_requirement_items;
DROP TABLE IF EXISTS additional_requirements;
DROP TYPE IF EXISTS direct_purchase_type;
