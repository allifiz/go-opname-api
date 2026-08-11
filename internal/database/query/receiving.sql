-- name: CreateReceipt :one
INSERT INTO receipts (
    purchase_order_id,
    received_by,
    note
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetReceiptByID :one
SELECT *
FROM receipts
WHERE id = $1;

-- name: ListReceiptsByPurchaseOrder :many
SELECT *
FROM receipts
WHERE purchase_order_id = $1
ORDER BY received_at ASC, created_at ASC;

-- name: CreateReceiptItem :one
INSERT INTO receipt_items (
    receipt_id,
    purchase_order_item_id,
    material_id,
    received_qty,
    unit_id,
    agreed_unit_price
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListReceiptItems :many
SELECT
    ri.id,
    ri.receipt_id,
    ri.purchase_order_item_id,
    ri.material_id,
    m.name AS material_name,
    ri.received_qty,
    ri.unit_id,
    u.code AS unit_code,
    ri.agreed_unit_price,
    ri.actual_amount,
    ri.created_at
FROM receipt_items ri
JOIN materials m ON m.id = ri.material_id
JOIN units u ON u.id = ri.unit_id
WHERE ri.receipt_id = $1
ORDER BY m.name ASC;

-- name: SumReceivedQtyByPurchaseOrderItem :one
SELECT COALESCE(SUM(received_qty), 0)::NUMERIC(18,4) AS total_received_qty
FROM receipt_items
WHERE purchase_order_item_id = $1;

-- name: LockPurchaseOrderItemForReceipt :one
SELECT
    poi.*,
    po.status AS purchase_order_status
FROM purchase_order_items poi
JOIN purchase_orders po ON po.id = poi.purchase_order_id
WHERE poi.id = $1
FOR UPDATE OF poi;

-- name: UpdatePurchaseOrderItemReceiptStatus :one
UPDATE purchase_order_items
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CalculatePurchaseOrderReceiptStatus :one
SELECT CASE
    WHEN COUNT(*) FILTER (
        WHERE poi.status NOT IN ('CANCELLED', 'RECEIVED', 'OVER_RECEIVED', 'FULFILLED')
    ) = 0 THEN 'COMPLETED'::purchase_order_status
    WHEN COUNT(*) FILTER (
        WHERE poi.status IN ('PARTIAL_RECEIVED', 'RECEIVED', 'OVER_RECEIVED', 'FULFILLED')
    ) > 0 THEN 'PARTIALLY_RECEIVED'::purchase_order_status
    ELSE 'VERIFIED'::purchase_order_status
END AS status
FROM purchase_order_items poi
WHERE poi.purchase_order_id = $1;

-- name: CreateReceiptDocument :one
INSERT INTO receipt_documents (
    receipt_id,
    document_type,
    file_url,
    file_name,
    uploaded_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListReceiptDocuments :many
SELECT *
FROM receipt_documents
WHERE receipt_id = $1
ORDER BY created_at ASC;
