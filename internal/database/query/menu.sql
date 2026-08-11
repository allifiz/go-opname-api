-- name: CreatePeriod :one
INSERT INTO periods (
    name,
    start_date,
    end_date,
    created_by
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPeriodByID :one
SELECT *
FROM periods
WHERE id = $1;

-- name: ListPeriods :many
SELECT *
FROM periods
ORDER BY start_date DESC, created_at DESC;

-- name: CreateMenuTemplate :one
INSERT INTO menu_templates (
    name,
    description,
    created_by
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMenuTemplateByID :one
SELECT *
FROM menu_templates
WHERE id = $1;

-- name: ListMenuTemplates :many
SELECT *
FROM menu_templates
ORDER BY name ASC;

-- name: UpdateMenuTemplate :one
UPDATE menu_templates
SET
    name = $2,
    description = $3,
    is_active = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateMenuTemplateComponent :one
INSERT INTO menu_template_components (
    menu_template_id,
    name,
    sort_order
) VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMenuTemplateComponents :many
SELECT *
FROM menu_template_components
WHERE menu_template_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: CreateMenuTemplateComponentMaterial :one
INSERT INTO menu_template_component_materials (
    menu_template_component_id,
    material_id,
    qty_per_portion,
    unit_id
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMenuTemplateComponentMaterials :many
SELECT
    mtcm.id,
    mtcm.menu_template_component_id,
    mtcm.material_id,
    m.name AS material_name,
    mtcm.qty_per_portion,
    mtcm.unit_id,
    u.code AS unit_code,
    mtcm.created_at,
    mtcm.updated_at
FROM menu_template_component_materials mtcm
JOIN materials m ON m.id = mtcm.material_id
JOIN units u ON u.id = mtcm.unit_id
WHERE mtcm.menu_template_component_id = $1
ORDER BY m.name ASC;

-- name: CreateScheduledMenu :one
INSERT INTO scheduled_menus (
    period_id,
    menu_template_id,
    menu_date,
    planned_portions,
    created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetScheduledMenuByID :one
SELECT *
FROM scheduled_menus
WHERE id = $1;

-- name: ListScheduledMenusByPeriod :many
SELECT *
FROM scheduled_menus
WHERE period_id = $1
ORDER BY menu_date ASC;

-- name: CreateScheduledMenuComponent :one
INSERT INTO scheduled_menu_components (
    scheduled_menu_id,
    source_template_component_id,
    name,
    sort_order
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListScheduledMenuComponents :many
SELECT *
FROM scheduled_menu_components
WHERE scheduled_menu_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: CreateScheduledMenuComponentMaterial :one
INSERT INTO scheduled_menu_component_materials (
    scheduled_menu_component_id,
    source_template_material_id,
    material_id,
    qty_per_portion,
    unit_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListScheduledMenuComponentMaterials :many
SELECT
    smcm.id,
    smcm.scheduled_menu_component_id,
    smcm.source_template_material_id,
    smcm.material_id,
    m.name AS material_name,
    smcm.qty_per_portion,
    smcm.unit_id,
    u.code AS unit_code,
    smcm.created_at,
    smcm.updated_at
FROM scheduled_menu_component_materials smcm
JOIN materials m ON m.id = smcm.material_id
JOIN units u ON u.id = smcm.unit_id
WHERE smcm.scheduled_menu_component_id = $1
ORDER BY m.name ASC;

-- name: GetScheduledMenuGrossRequirements :many
SELECT
    smcm.material_id,
    m.name AS material_name,
    smcm.unit_id,
    u.code AS unit_code,
    SUM(smcm.qty_per_portion * sm.planned_portions)::NUMERIC(18,4) AS gross_requirement_qty
FROM scheduled_menus sm
JOIN scheduled_menu_components smc ON smc.scheduled_menu_id = sm.id
JOIN scheduled_menu_component_materials smcm ON smcm.scheduled_menu_component_id = smc.id
JOIN materials m ON m.id = smcm.material_id
JOIN units u ON u.id = smcm.unit_id
WHERE sm.id = $1
GROUP BY smcm.material_id, m.name, smcm.unit_id, u.code
ORDER BY m.name ASC;

-- name: GetScheduledMenuPerPortionRequirements :many
SELECT
    smcm.material_id,
    m.name AS material_name,
    smcm.unit_id,
    u.code AS unit_code,
    SUM(smcm.qty_per_portion)::NUMERIC(18,4) AS qty_per_portion
FROM scheduled_menu_components smc
JOIN scheduled_menu_component_materials smcm ON smcm.scheduled_menu_component_id = smc.id
JOIN materials m ON m.id = smcm.material_id
JOIN units u ON u.id = smcm.unit_id
WHERE smc.scheduled_menu_id = $1
GROUP BY smcm.material_id, m.name, smcm.unit_id, u.code
ORDER BY m.name ASC;
