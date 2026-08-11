-- +goose Up
CREATE TYPE material_usage_status AS ENUM (
    'DRAFT',
    'WAITING_APPROVAL',
    'APPROVED',
    'NEEDS_REVISION'
);

CREATE TYPE approval_decision AS ENUM (
    'APPROVED',
    'REJECTED'
);

CREATE TYPE usage_approver_role AS ENUM (
    'CHEF',
    'AKUNTAN'
);

CREATE TABLE material_usages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL UNIQUE REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    usage_date DATE NOT NULL,
    submitted_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    status material_usage_status NOT NULL DEFAULT 'DRAFT',
    version INTEGER NOT NULL DEFAULT 1,
    submitted_at TIMESTAMPTZ,
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_material_usage_version_positive CHECK (version > 0),
    CONSTRAINT chk_material_usage_submit_lifecycle CHECK (
        (status = 'DRAFT' AND submitted_at IS NULL)
        OR (status IN ('WAITING_APPROVAL', 'NEEDS_REVISION') AND submitted_at IS NOT NULL)
        OR (status = 'APPROVED' AND submitted_at IS NOT NULL AND applied_at IS NOT NULL)
    )
);

CREATE INDEX idx_material_usages_status ON material_usages(status);

CREATE TABLE material_usage_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_usage_id UUID NOT NULL REFERENCES material_usages(id) ON UPDATE CASCADE ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    planned_qty NUMERIC(18,4) NOT NULL,
    actual_qty NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_material_usage_planned_non_negative CHECK (planned_qty >= 0),
    CONSTRAINT chk_material_usage_actual_non_negative CHECK (actual_qty >= 0),
    CONSTRAINT uq_material_usage_material UNIQUE (material_usage_id, material_id)
);

CREATE TABLE material_usage_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    material_usage_id UUID NOT NULL REFERENCES material_usages(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    approver_role usage_approver_role NOT NULL,
    approver_id UUID NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    entity_version INTEGER NOT NULL,
    status approval_decision NOT NULL,
    note TEXT,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_material_usage_approval_version_positive CHECK (entity_version > 0),
    CONSTRAINT uq_material_usage_approval_role_version UNIQUE (material_usage_id, approver_role, entity_version)
);

CREATE INDEX idx_material_usage_approvals_usage_version
    ON material_usage_approvals(material_usage_id, entity_version);

-- +goose Down
DROP TABLE IF EXISTS material_usage_approvals;
DROP TABLE IF EXISTS material_usage_items;
DROP TABLE IF EXISTS material_usages;
DROP TYPE IF EXISTS usage_approver_role;
DROP TYPE IF EXISTS approval_decision;
DROP TYPE IF EXISTS material_usage_status;
