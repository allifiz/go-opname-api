-- +goose Up
CREATE TYPE stock_opname_status AS ENUM (
    'DRAFT',
    'MATCHED',
    'DIFFERENCE_FOUND',
    'WAITING_ADJUSTMENT_APPROVAL',
    'COMPLETED'
);

CREATE TYPE stock_adjustment_status AS ENUM (
    'DRAFT',
    'WAITING_APPROVAL',
    'APPROVED',
    'NEEDS_REVISION'
);

CREATE TYPE stock_adjustment_approver_role AS ENUM ('CHEF', 'AKUNTAN');
CREATE TYPE stock_adjustment_approval_decision AS ENUM ('APPROVED', 'REJECTED');

CREATE TABLE stock_opnames (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL UNIQUE REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    opname_date DATE NOT NULL,
    performed_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    status stock_opname_status NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE stock_opname_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_opname_id UUID NOT NULL REFERENCES stock_opnames(id) ON UPDATE CASCADE ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    system_qty NUMERIC(18,4) NOT NULL,
    physical_qty NUMERIC(18,4) NOT NULL,
    difference_qty NUMERIC(18,4) GENERATED ALWAYS AS (physical_qty - system_qty) STORED,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_opname_system_qty_non_negative CHECK (system_qty >= 0),
    CONSTRAINT chk_stock_opname_physical_qty_non_negative CHECK (physical_qty >= 0),
    CONSTRAINT uq_stock_opname_material UNIQUE (stock_opname_id, material_id)
);

CREATE TABLE stock_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_opname_item_id UUID NOT NULL UNIQUE REFERENCES stock_opname_items(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    adjustment_qty NUMERIC(18,4) NOT NULL,
    reason TEXT NOT NULL,
    submitted_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    status stock_adjustment_status NOT NULL DEFAULT 'DRAFT',
    version INTEGER NOT NULL DEFAULT 1,
    submitted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_adjustment_qty_non_zero CHECK (adjustment_qty <> 0),
    CONSTRAINT chk_stock_adjustment_reason_not_blank CHECK (BTRIM(reason) <> ''),
    CONSTRAINT chk_stock_adjustment_version_positive CHECK (version > 0)
);

CREATE TABLE stock_adjustment_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_adjustment_id UUID NOT NULL REFERENCES stock_adjustments(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    approver_role stock_adjustment_approver_role NOT NULL,
    approver_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    entity_version INTEGER NOT NULL,
    status stock_adjustment_approval_decision NOT NULL,
    note TEXT,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_stock_adjustment_approval_version_positive CHECK (entity_version > 0),
    CONSTRAINT uq_stock_adjustment_approval_role_version UNIQUE (stock_adjustment_id, approver_role, entity_version)
);

CREATE INDEX idx_stock_opname_items_material ON stock_opname_items(material_id);
CREATE INDEX idx_stock_adjustment_approvals_adjustment_version ON stock_adjustment_approvals(stock_adjustment_id, entity_version);

-- +goose Down
DROP TABLE IF EXISTS stock_adjustment_approvals;
DROP TABLE IF EXISTS stock_adjustments;
DROP TABLE IF EXISTS stock_opname_items;
DROP TABLE IF EXISTS stock_opnames;
DROP TYPE IF EXISTS stock_adjustment_approval_decision;
DROP TYPE IF EXISTS stock_adjustment_approver_role;
DROP TYPE IF EXISTS stock_adjustment_status;
DROP TYPE IF EXISTS stock_opname_status;
