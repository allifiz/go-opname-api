-- name: GetMaterialStock :one
SELECT
    ms.material_id,
    m.name AS material_name,
    ms.qty,
    ms.unit_id,
    u.code AS unit_code,
    ms.updated_at
FROM material_stocks ms
JOIN materials m ON m.id = ms.material_id
JOIN units u ON u.id = ms.unit_id
WHERE ms.material_id = $1;

-- name: LockMaterialStock :one
SELECT material_id, qty, unit_id, updated_at
FROM material_stocks
WHERE material_id = $1
FOR UPDATE;

-- name: UpsertMaterialStock :one
INSERT INTO material_stocks (
    material_id,
    qty,
    unit_id
) VALUES ($1, $2, $3)
ON CONFLICT (material_id) DO UPDATE
SET
    qty = EXCLUDED.qty,
    unit_id = EXCLUDED.unit_id,
    updated_at = NOW()
RETURNING *;

-- name: SumActiveReservedStockByMaterial :one
SELECT COALESCE(SUM(qty), 0)::NUMERIC(18,4) AS reserved_qty
FROM stock_reservations
WHERE material_id = $1
  AND status = 'ACTIVE';

-- name: CreateStockReservation :one
INSERT INTO stock_reservations (
    scheduled_menu_id,
    procurement_request_item_id,
    material_id,
    qty,
    unit_id,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetStockReservationByID :one
SELECT *
FROM stock_reservations
WHERE id = $1;

-- name: ListActiveStockReservationsByMaterial :many
SELECT *
FROM stock_reservations
WHERE material_id = $1
  AND status = 'ACTIVE'
ORDER BY reserved_at ASC;

-- name: ReleaseStockReservation :one
UPDATE stock_reservations
SET
    status = 'RELEASED',
    released_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'ACTIVE'
RETURNING *;

-- name: ConsumeStockReservation :one
UPDATE stock_reservations
SET
    status = 'CONSUMED',
    consumed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'ACTIVE'
RETURNING *;

-- name: CreateStockMovement :one
INSERT INTO stock_movements (
    material_id,
    movement_type,
    reference_type,
    reference_id,
    qty,
    unit_id,
    movement_date,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListStockMovementsByMaterial :many
SELECT *
FROM stock_movements
WHERE material_id = $1
ORDER BY movement_date DESC, created_at DESC;
