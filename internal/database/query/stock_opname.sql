-- name: CreateStockOpname :one
INSERT INTO stock_opnames (
    scheduled_menu_id,
    opname_date,
    performed_by
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetStockOpnameByID :one
SELECT *
FROM stock_opnames
WHERE id = $1;

-- name: GetStockOpnameByScheduledMenu :one
SELECT *
FROM stock_opnames
WHERE scheduled_menu_id = $1;

-- name: LockStockOpname :one
SELECT *
FROM stock_opnames
WHERE id = $1
FOR UPDATE;

-- name: CreateStockOpnameItem :one
INSERT INTO stock_opname_items (
    stock_opname_id,
    material_id,
    system_qty,
    physical_qty,
    unit_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetStockOpnameItemByID :one
SELECT *
FROM stock_opname_items
WHERE id = $1;

-- name: UpdateStockOpnameItemPhysicalQty :one
UPDATE stock_opname_items
SET physical_qty = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListStockOpnameItems :many
SELECT
    soi.id,
    soi.stock_opname_id,
    soi.material_id,
    m.name AS material_name,
    soi.system_qty,
    soi.physical_qty,
    soi.difference_qty,
    soi.unit_id,
    u.code AS unit_code,
    soi.created_at,
    soi.updated_at
FROM stock_opname_items soi
JOIN materials m ON m.id = soi.material_id
JOIN units u ON u.id = soi.unit_id
WHERE soi.stock_opname_id = $1
ORDER BY m.name ASC;

-- name: UpdateStockOpnameStatus :one
UPDATE stock_opnames
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateStockAdjustment :one
INSERT INTO stock_adjustments (
    stock_opname_item_id,
    material_id,
    adjustment_qty,
    reason,
    submitted_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetStockAdjustmentByID :one
SELECT *
FROM stock_adjustments
WHERE id = $1;

-- name: GetStockAdjustmentByOpnameItem :one
SELECT *
FROM stock_adjustments
WHERE stock_opname_item_id = $1;

-- name: ListStockAdjustmentsByOpname :many
SELECT sa.*
FROM stock_adjustments sa
JOIN stock_opname_items soi ON soi.id = sa.stock_opname_item_id
WHERE soi.stock_opname_id = $1
ORDER BY sa.created_at ASC;

-- name: LockStockAdjustment :one
SELECT
    sa.*,
    soi.stock_opname_id,
    soi.unit_id,
    soi.system_qty,
    soi.physical_qty,
    so.scheduled_menu_id
FROM stock_adjustments sa
JOIN stock_opname_items soi ON soi.id = sa.stock_opname_item_id
JOIN stock_opnames so ON so.id = soi.stock_opname_id
WHERE sa.id = $1
FOR UPDATE OF sa, soi;

-- name: UpdateStockAdjustmentForRevision :one
UPDATE stock_adjustments
SET
    adjustment_qty = $2,
    reason = $3,
    submitted_by = $4,
    status = 'DRAFT',
    version = version + 1,
    submitted_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status IN ('DRAFT', 'NEEDS_REVISION')
RETURNING *;

-- name: SubmitStockAdjustment :one
UPDATE stock_adjustments
SET
    submitted_by = $2,
    status = 'WAITING_APPROVAL',
    submitted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'DRAFT'
RETURNING *;

-- name: MarkStockAdjustmentNeedsRevision :one
UPDATE stock_adjustments
SET status = 'NEEDS_REVISION', updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_APPROVAL'
RETURNING *;

-- name: MarkStockAdjustmentApproved :one
UPDATE stock_adjustments
SET status = 'APPROVED', updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_APPROVAL'
RETURNING *;

-- name: CreateStockAdjustmentApproval :one
INSERT INTO stock_adjustment_approvals (
    stock_adjustment_id,
    approver_role,
    approver_id,
    entity_version,
    status,
    note
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListStockAdjustmentApprovals :many
SELECT *
FROM stock_adjustment_approvals
WHERE stock_adjustment_id = $1
ORDER BY entity_version ASC, decided_at ASC;

-- name: CountApprovedStockAdjustmentRolesForVersion :one
SELECT COUNT(DISTINCT approver_role)::INT AS approved_count
FROM stock_adjustment_approvals
WHERE stock_adjustment_id = $1
  AND entity_version = $2
  AND status = 'APPROVED';

-- name: CountOpenStockAdjustmentsByOpname :one
SELECT COUNT(*)::INT AS open_count
FROM stock_adjustments sa
JOIN stock_opname_items soi ON soi.id = sa.stock_opname_item_id
WHERE soi.stock_opname_id = $1
  AND sa.status <> 'APPROVED';
