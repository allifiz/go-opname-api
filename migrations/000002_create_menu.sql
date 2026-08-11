-- +goose Up
CREATE TABLE periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_period_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_period_two_weeks CHECK (end_date = start_date + 13)
);

CREATE INDEX idx_periods_date_range ON periods(start_date, end_date);

CREATE TABLE menu_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_menu_templates_is_active ON menu_templates(is_active);
CREATE INDEX idx_menu_templates_name ON menu_templates(name);

CREATE TABLE menu_template_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_template_id UUID NOT NULL REFERENCES menu_templates(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_menu_template_component_sort_order CHECK (sort_order >= 0)
);

CREATE INDEX idx_menu_template_components_template_id
    ON menu_template_components(menu_template_id);

CREATE TABLE menu_template_component_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_template_component_id UUID NOT NULL REFERENCES menu_template_components(id) ON UPDATE CASCADE ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    qty_per_portion NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_menu_template_material_qty CHECK (qty_per_portion > 0),
    CONSTRAINT uq_menu_template_component_material UNIQUE (menu_template_component_id, material_id)
);

CREATE INDEX idx_menu_template_component_materials_material_id
    ON menu_template_component_materials(material_id);

CREATE TABLE scheduled_menus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id UUID NOT NULL REFERENCES periods(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    menu_template_id UUID REFERENCES menu_templates(id) ON UPDATE CASCADE ON DELETE SET NULL,
    menu_date DATE NOT NULL,
    planned_portions INTEGER NOT NULL,
    created_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scheduled_menu_portions CHECK (planned_portions > 0),
    CONSTRAINT uq_scheduled_menu_period_date UNIQUE (period_id, menu_date)
);

CREATE INDEX idx_scheduled_menus_period_id ON scheduled_menus(period_id);
CREATE INDEX idx_scheduled_menus_menu_date ON scheduled_menus(menu_date);

CREATE TABLE scheduled_menu_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_id UUID NOT NULL REFERENCES scheduled_menus(id) ON UPDATE CASCADE ON DELETE CASCADE,
    source_template_component_id UUID REFERENCES menu_template_components(id) ON UPDATE CASCADE ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scheduled_menu_component_sort_order CHECK (sort_order >= 0)
);

CREATE INDEX idx_scheduled_menu_components_menu_id
    ON scheduled_menu_components(scheduled_menu_id);

CREATE TABLE scheduled_menu_component_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_menu_component_id UUID NOT NULL REFERENCES scheduled_menu_components(id) ON UPDATE CASCADE ON DELETE CASCADE,
    source_template_material_id UUID REFERENCES menu_template_component_materials(id) ON UPDATE CASCADE ON DELETE SET NULL,
    material_id UUID NOT NULL REFERENCES materials(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    qty_per_portion NUMERIC(18,4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scheduled_menu_material_qty CHECK (qty_per_portion > 0),
    CONSTRAINT uq_scheduled_menu_component_material UNIQUE (scheduled_menu_component_id, material_id)
);

CREATE INDEX idx_scheduled_menu_component_materials_material_id
    ON scheduled_menu_component_materials(material_id);

-- +goose Down
DROP TABLE IF EXISTS scheduled_menu_component_materials;
DROP TABLE IF EXISTS scheduled_menu_components;
DROP TABLE IF EXISTS scheduled_menus;
DROP TABLE IF EXISTS menu_template_component_materials;
DROP TABLE IF EXISTS menu_template_components;
DROP TABLE IF EXISTS menu_templates;
DROP TABLE IF EXISTS periods;
