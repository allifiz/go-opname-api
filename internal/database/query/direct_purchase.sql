-- name: LockScheduledMenuForAdditionalRequirement :one
SELECT *
FROM scheduled_menus
WHERE id = $1
FOR UPDATE;

-- name: GetLatestAdditionalRequirementByScheduledMenu :one
SELECT *
FROM additional_requirements
WHERE scheduled_menu_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: LockPurchaseOrderItemForShortagePurchase :one
SELECT
    poi.*,
    po.scheduled_menu_id,
    po.status AS purchase_order_status
FROM purchase_order_items poi
JOIN purchase_orders po ON po.id = poi.purchase_order_id
WHERE poi.id = $1
FOR UPDATE OF poi;

-- name: CreateAdditionalRequirement :one
INSERT INTO additional_requirements (
    scheduled_menu_id,
    previous_portions,
    new_portions,
    created_by
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateAdditionalRequirementItem :one
INSERT INTO additional_requirement_items (
    additional_requirement_id,
    material_id,
    additional_qty,
    unit_id
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAdditionalRequirementItems :many
SELECT
    ari.id,
    ari.additional_requirement_id,
    ari.material_id,
    m.name AS material_name,
    ari.additional_qty,
    ari.unit_id,
    u.code AS unit_code,
    ari.created_at
FROM additional_requirement_items ari
JOIN materials m ON m.id = ari.material_id
JOIN units u ON u.id = ari.unit_id
WHERE ari.additional_requirement_id = $1
ORDER BY m.name ASC;

-- name: CreateDirectPurchase :one
INSERT INTO direct_purchases (
    scheduled_menu_id,
    purchase_type,
    source_name,
    purchased_by,
    note
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateDirectPurchaseItem :one
INSERT INTO direct_purchase_items (
    direct_purchase_id,
    purchase_order_item_id,
    additional_requirement_item_id,
    material_id,
    qty,
    unit_id,
    unit_price
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDirectPurchaseByID :one
SELECT *
FROM direct_purchases
WHERE id = $1;

-- name: ListDirectPurchasesByScheduledMenu :many
SELECT *
FROM direct_purchases
WHERE scheduled_menu_id = $1
ORDER BY purchase_date DESC, created_at DESC;

-- name: ListDirectPurchaseItems :many
SELECT
    dpi.id,
    dpi.direct_purchase_id,
    dpi.purchase_order_item_id,
    dpi.additional_requirement_item_id,
    dpi.material_id,
    m.name AS material_name,
    dpi.qty,
    dpi.unit_id,
    u.code AS unit_code,
    dpi.unit_price,
    dpi.total_amount,
    dpi.created_at
FROM direct_purchase_items dpi
JOIN materials m ON m.id = dpi.material_id
JOIN units u ON u.id = dpi.unit_id
WHERE dpi.direct_purchase_id = $1
ORDER BY m.name ASC;

-- name: SumShortageDirectPurchaseQtyByPOItem :one
SELECT COALESCE(SUM(dpi.qty), 0)::NUMERIC(18,4) AS purchased_qty
FROM direct_purchase_items dpi
JOIN direct_purchases dp ON dp.id = dpi.direct_purchase_id
WHERE dpi.purchase_order_item_id = $1
  AND dp.purchase_type = 'SHORTAGE';
