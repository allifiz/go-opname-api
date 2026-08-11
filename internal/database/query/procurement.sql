-- name: CreateProcurementRequest :one
INSERT INTO procurement_requests (
    scheduled_menu_id,
    checked_by,
    checked_at
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProcurementRequestByID :one
SELECT *
FROM procurement_requests
WHERE id = $1;

-- name: ListProcurementRequestsByScheduledMenu :many
SELECT *
FROM procurement_requests
WHERE scheduled_menu_id = $1
ORDER BY created_at DESC;

-- name: CreateProcurementRequestItem :one
INSERT INTO procurement_request_items (
    procurement_request_id,
    material_id,
    gross_requirement_qty,
    existing_stock_qty,
    reserved_stock_qty,
    net_procurement_qty,
    unit_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListProcurementRequestItems :many
SELECT
    pri.id,
    pri.procurement_request_id,
    pri.material_id,
    m.name AS material_name,
    pri.gross_requirement_qty,
    pri.existing_stock_qty,
    pri.reserved_stock_qty,
    pri.net_procurement_qty,
    pri.unit_id,
    u.code AS unit_code,
    pri.created_at,
    pri.updated_at
FROM procurement_request_items pri
JOIN materials m ON m.id = pri.material_id
JOIN units u ON u.id = pri.unit_id
WHERE pri.procurement_request_id = $1
ORDER BY m.name ASC;

-- name: SubmitProcurementRequest :one
UPDATE procurement_requests
SET
    status = 'WAITING_VERIFICATION',
    submitted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status IN ('DRAFT', 'REJECTED')
RETURNING *;

-- name: VerifyProcurementRequest :one
UPDATE procurement_requests
SET
    status = 'VERIFIED',
    verified_by = $2,
    verified_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_VERIFICATION'
RETURNING *;

-- name: RejectProcurementRequest :one
UPDATE procurement_requests
SET
    status = 'REJECTED',
    verified_by = NULL,
    verified_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING_VERIFICATION'
RETURNING *;

-- name: CreatePurchaseOrder :one
INSERT INTO purchase_orders (
    procurement_request_id,
    scheduled_menu_id,
    po_number,
    delivery_date,
    status,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPurchaseOrderByID :one
SELECT *
FROM purchase_orders
WHERE id = $1;

-- name: GetPurchaseOrderByProcurementRequest :one
SELECT *
FROM purchase_orders
WHERE procurement_request_id = $1;

-- name: ListPurchaseOrdersByScheduledMenu :many
SELECT *
FROM purchase_orders
WHERE scheduled_menu_id = $1
ORDER BY delivery_date ASC, created_at ASC;

-- name: CreatePurchaseOrderItem :one
INSERT INTO purchase_order_items (
    purchase_order_id,
    procurement_request_item_id,
    material_id,
    ordered_qty,
    unit_id,
    agreed_unit_price,
    supplier_name
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListPurchaseOrderItems :many
SELECT
    poi.id,
    poi.purchase_order_id,
    poi.procurement_request_item_id,
    poi.material_id,
    m.name AS material_name,
    poi.ordered_qty,
    poi.unit_id,
    u.code AS unit_code,
    poi.agreed_unit_price,
    poi.supplier_name,
    poi.status,
    poi.cancelled_at,
    poi.cancelled_by,
    poi.cancel_reason,
    poi.created_at,
    poi.updated_at
FROM purchase_order_items poi
JOIN materials m ON m.id = poi.material_id
JOIN units u ON u.id = poi.unit_id
WHERE poi.purchase_order_id = $1
ORDER BY m.name ASC;

-- name: LockPurchaseOrderItemForCancellation :one
SELECT
    poi.id,
    poi.purchase_order_id,
    poi.procurement_request_item_id,
    poi.material_id,
    poi.ordered_qty,
    poi.unit_id,
    poi.status,
    po.scheduled_menu_id,
    po.delivery_date
FROM purchase_order_items poi
JOIN purchase_orders po ON po.id = poi.purchase_order_id
WHERE poi.id = $1
FOR UPDATE OF poi, po;

-- name: CancelPurchaseOrderItem :one
UPDATE purchase_order_items
SET
    status = 'CANCELLED',
    cancelled_at = NOW(),
    cancelled_by = $2,
    cancel_reason = $3,
    updated_at = NOW()
WHERE id = $1
  AND status = 'WAITING'
RETURNING *;

-- name: UpdatePurchaseOrderStatus :one
UPDATE purchase_orders
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
