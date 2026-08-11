-- name: CreateMaterialUsage :one
INSERT INTO material_usages (
    scheduled_menu_id,
    usage_date,
    submitted_by
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMaterialUsageByID :one
SELECT *
FROM material_usages
WHERE id = $1;

-- name: GetMaterialUsageByScheduledMenu :one
SELECT *
FROM material_usages
WHERE scheduled_menu_id = $1;

-- name: LockMaterialUsage :one
SELECT *
FROM material_usages
WHERE id = $1
FOR UPDATE;

-- name: CreateMaterialUsageItem :one
INSERT INTO material_usage_items (
    material_usage_id,
    material_id,
    planned_qty,
    actual_qty,
    unit_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteMaterialUsageItems :exec
DELETE FROM material_usage_items
WHERE material_usage_id = $1;

-- name: ListMaterialUsageItems :many
SELECT
    mui.id,
    mui.material_usage_id,
    mui.material_id,
    m.name AS material_name,
    mui.planned_qty,
    mui.actual_qty,
    mui.unit_id,
    u.code AS unit_code,
    mui.created_at,
    mui.updated_at
FROM material_usage_items mui
JOIN materials m ON m.id = mui.material_id
JOIN units u ON u.id = mui.unit_id
WHERE mui.material_usage_id = $1
ORDER BY m.name ASC;

-- name: UpdateMaterialUsageForRevision :one
UPDATE material_usages
SET
    usage_date = $2,
    submitted_by = $3,
    status = 'DRAFT',
    version = version + 1,
    submitted_at = NULL,
    applied_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status IN ('DRAFT', 'NEEDS_REVISION')
RETURNING *;

-- name: SubmitMaterialUsage :one
UPDATE material_usages
SET
    status = 'WAITING_APPROVAL',
    submitted_by = $2,
    submitted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'DRAFT'
RETURNING *;

-- name: CreateMaterialUsageApproval :one
INSERT INTO material_usage_approvals (
    material_usage_id,
    approver_role,
    approver_id,
    entity_version,
    status,
    note
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListMaterialUsageApprovals :many
SELECT *
FROM material_usage_approvals
WHERE material_usage_id = $1
ORDER BY entity_version ASC, decided_at ASC;

-- name: CountApprovedMaterialUsageRolesForVersion :one
SELECT COUNT(DISTINCT approver_role)::INT AS approved_count
FROM material_usage_approvals
WHERE material_usage_id = $1
  AND entity_version = $2
  AND status = 'APPROVED';

-- name: MarkMaterialUsageNeedsRevision :one
UPDATE material_usages
SET
    status = 'NEEDS_REVISION',
    updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_APPROVAL'
RETURNING *;

-- name: MarkMaterialUsageApproved :one
UPDATE material_usages
SET
    status = 'APPROVED',
    applied_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_APPROVAL'
RETURNING *;
