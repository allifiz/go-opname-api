# Database Design V1

## Migration Plan

```text
000001_create_foundation.sql             IMPLEMENTED
000002_create_menu.sql                   IMPLEMENTED
000003_create_inventory.sql              IMPLEMENTED
000004_create_procurement.sql            IMPLEMENTED
000005_purchase_order_constraints.sql    IMPLEMENTED
000006_create_receiving.sql              IMPLEMENTED
000007_direct_purchase.sql               NEXT
000008_material_usage.sql
000009_stock_opname.sql
```

## General Conventions
- Primary keys: `UUID DEFAULT gen_random_uuid()`.
- Quantities: `NUMERIC(18,4)`.
- Money: `NUMERIC(18,2)`.
- Operational timestamps: `TIMESTAMPTZ`.
- Production/menu dates: `DATE`.
- Negative stock is forbidden.

## Foundation / Menu / Inventory / Procurement
Foundation, reusable menu templates, scheduled-menu snapshots, stock ledger/reservations, procurement requests, PO generation, and H-1 cancellation are implemented.

Core procurement allocation remains:
```text
active_reserved_stock = SUM(ACTIVE reservation qty)
available_stock = MAX(current_stock - active_reserved_stock, 0)
allocation_qty = MIN(gross_requirement, available_stock)
net_procurement = MAX(gross_requirement - available_stock, 0)
```

PO item `ordered_qty` is copied from verified positive `net_procurement_qty`. H-1 cancellation reserves replacement stock before marking the PO item `CANCELLED`.

## 000006 Receiving

### receipts
Receipt header fields:
- `purchase_order_id`
- `received_at`
- `received_by`
- `note`
- timestamps

A PO may have multiple receipts.

### receipt_items
Each row links one receipt event to one PO item.

Important fields:
- `receipt_id`
- `purchase_order_item_id`
- `material_id`
- `received_qty NUMERIC(18,4)`
- `unit_id`
- `agreed_unit_price NUMERIC(18,2)`
- generated `actual_amount NUMERIC(18,2)`

Rules:
- `received_qty >= 0`.
- Zero is valid and records a vendor non-delivery event (`NOT_RECEIVED`).
- Positive quantity enters stock.
- Material/unit/price are copied from the PO item, not supplied by the client.
- One PO item can appear only once inside the same receipt, but may appear again in later receipts.
- `actual_amount = ROUND(received_qty * agreed_unit_price, 2)` is PostgreSQL-generated.

### receipt_documents
Document types:
- `INVOICE`
- `NOTA`
- `PHOTO`
- `OTHER`

The database stores file metadata/path/URL, not file blobs.

## Cumulative Receiving Status
PO item status is based on all vendor receipt rows for that PO item:

```text
total_received_qty = SUM(receipt_items.received_qty)

0                           -> NOT_RECEIVED
0 < total < ordered_qty     -> PARTIAL_RECEIVED
total = ordered_qty         -> RECEIVED
total > ordered_qty         -> OVER_RECEIVED
```

Over-delivery is therefore cumulative, not calculated independently per partial receipt.

PO header status is recalculated from item statuses:
- all items still open/no receipt progress -> `VERIFIED`;
- some receiving progress but at least one item remains open -> `PARTIALLY_RECEIVED`;
- every item is terminal (`CANCELLED`, `RECEIVED`, `OVER_RECEIVED`, `FULFILLED`) -> `COMPLETED`.

## Receiving Transaction
Each receipt is atomic:

```text
create receipt header
for each input PO item:
    lock PO item FOR UPDATE
    validate item belongs to target PO
    create receipt item
    if received_qty > 0:
        ensure + lock material stock
        increase material stock
        create stock movement IN / PO_RECEIPT
    sum cumulative received qty
    update PO item status
create receipt document metadata
recalculate PO header status
commit
```

The PO-item lock serializes concurrent receipt submissions for the same item, preventing cumulative-status races.

A zero-quantity receipt does not create a stock movement because no physical inventory changed.

## sqlc Query Files
Implemented:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`
- `internal/database/query/receiving.sql`

Receiving adds queries for receipt CRUD/read, cumulative receipt sums, PO-item locking/status updates, PO header status calculation, document metadata, and atomic stock increase.

## Planned Direct Purchase
Next migration: `000007_direct_purchase.sql`.

Types:
- `SHORTAGE`
- `ADDITIONAL_REQUIREMENT`

Rules:
- `initial_shortage_qty = MAX(ordered_qty - total_vendor_received_qty, 0)`.
- `remaining_shortage_qty = MAX(initial_shortage_qty - total_shortage_purchase_qty, 0)`.
- `SHORTAGE` purchase may not exceed remaining shortage.
- Increased production after PO uses `ADDITIONAL_REQUIREMENT`, not a second PO.
- Direct purchase + stock IN must be atomic.

## Planned Usage / Opname
Material usage and stock adjustment both require Chef + Accountant approval. Stock changes occur only after valid approvals for the current entity version.

## Important Transaction Rules
- Procurement stock check + reservation: atomic with stock row lock.
- H-1 replacement reservation + PO-item cancellation: atomic.
- Receipt + stock IN: atomic with PO-item lock and cumulative status update.
- Direct purchase + stock IN: atomic.
- Approved usage + stock OUT: atomic.
- Approved adjustment + stock movement: atomic.
