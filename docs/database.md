# Database Design V1

## Migration Plan

```text
000001_create_foundation.sql             IMPLEMENTED
000002_create_menu.sql                   IMPLEMENTED
000003_create_inventory.sql              IMPLEMENTED
000004_create_procurement.sql            IMPLEMENTED
000005_purchase_order_constraints.sql    IMPLEMENTED
000006_create_receiving.sql              IMPLEMENTED
000007_create_direct_purchase.sql        IMPLEMENTED
000008_create_material_usage.sql         IMPLEMENTED
000009_create_stock_opname.sql           IMPLEMENTED
```

## General Conventions
- Primary keys: `UUID DEFAULT gen_random_uuid()`.
- Quantities: `NUMERIC(18,4)`.
- Money: `NUMERIC(18,2)`.
- Operational timestamps: `TIMESTAMPTZ`.
- Production/menu dates: `DATE`.
- Negative stock is forbidden.
- Real inventory changes must be represented in `stock_movements`.

## Implemented Core Flow
Foundation, reusable menu templates, scheduled snapshots, stock reservations, procurement, PO/H-1, receiving, direct purchase, material usage, and stock opname/adjustment are implemented.

## Material Usage
Usage is versioned and requires Chef + Accountant approval for the same current version before stock OUT. The second approval transaction applies stock decrement, `OUT / MATERIAL_USAGE` movements, reservation consumption, and final approval atomically.

## 000009 Stock Opname

### stock_opname_status
- `DRAFT`
- `MATCHED`
- `DIFFERENCE_FOUND`
- `WAITING_ADJUSTMENT_APPROVAL`
- `COMPLETED`

### stock_opnames
One record per scheduled menu in V1.

Important fields:
- `scheduled_menu_id UNIQUE`
- `opname_date`
- `performed_by`
- `status`
- timestamps

### stock_opname_items
For each scheduled-menu material:
- `system_qty NUMERIC(18,4) >= 0`
- `physical_qty NUMERIC(18,4) >= 0`
- generated `difference_qty = physical_qty - system_qty`
- `unit_id`

`system_qty` is captured from the locked current `material_stocks` row. Client controls only physical count and, when needed, adjustment reason.

A material is unique per opname.

### stock_adjustment_status
- `DRAFT`
- `WAITING_APPROVAL`
- `APPROVED`
- `NEEDS_REVISION`

### stock_adjustments
A stock adjustment exists only for a non-zero opname difference.

Fields:
- `stock_opname_item_id UNIQUE`
- `material_id`
- `adjustment_qty <> 0`
- `reason`
- `submitted_by`
- `status`
- `version > 0`
- `submitted_at`
- timestamps

`adjustment_qty` is server-derived from the opname item's generated `difference_qty`; it is not accepted directly from the client.

### stock_adjustment_approvals
Roles:
- `CHEF`
- `AKUNTAN`

Decisions:
- `APPROVED`
- `REJECTED`

Unique key:
```text
(stock_adjustment_id, approver_role, entity_version)
```

Historical approvals are retained across revisions. Only approvals matching the current adjustment version count.

## Opname Creation Transaction

```text
validate one opname per scheduled menu
resolve scheduled-menu material set
for each material:
    ensure material_stocks row
    lock material_stocks row
    snapshot system_qty
    persist physical_qty
    PostgreSQL generates difference_qty
    if difference != 0:
        require reason
        create DRAFT stock_adjustment using difference_qty
set opname MATCHED or DIFFERENCE_FOUND
commit
```

No stock is changed during this transaction.

## Adjustment Revision
Only `DRAFT` or `NEEDS_REVISION` may be edited.

Revision changes physical count rather than allowing a client-controlled adjustment number:
```text
lock adjustment + opname item
update physical_qty
recalculate generated difference_qty
require difference remains non-zero
copy difference into adjustment_qty
increment version
reset adjustment to DRAFT
keep older approvals as history
```

## Dual Approval + Adjustment Transaction
The second current-version approval applies inventory in the same transaction:

```text
lock stock_adjustment
validate WAITING_APPROVAL
validate active CHEF / AKUNTAN approver
insert versioned approval
if fewer than two current-version approvals:
    commit approval only
else:
    ensure + lock material_stocks
    if adjustment_qty > 0:
        increase stock
        create ADJUSTMENT_IN / STOCK_ADJUSTMENT movement
    if adjustment_qty < 0:
        require current stock >= abs(adjustment_qty)
        decrease stock
        create ADJUSTMENT_OUT / STOCK_ADJUSTMENT movement
    mark adjustment APPROVED
    if all opname adjustments are APPROVED:
        mark stock_opname COMPLETED
commit
```

If an `ADJUSTMENT_OUT` would make stock negative, the second approval and every inventory effect in that transaction roll back.

## sqlc Query Files
Implemented:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`
- `internal/database/query/receiving.sql`
- `internal/database/query/direct_purchase.sql`
- `internal/database/query/material_usage.sql`
- `internal/database/query/stock_opname.sql`

## Important Transaction Rules
- Procurement stock check + reservation: atomic with stock row lock.
- H-1 replacement reservation + PO-item cancellation: atomic.
- Receipt + stock IN: atomic with PO-item lock.
- SHORTAGE purchase + stock IN: atomic with PO-item lock.
- ADDITIONAL_REQUIREMENT + stock IN: atomic with scheduled-menu lock.
- Dual-approved material usage + stock OUT + reservation consumption: atomic.
- Dual-approved stock adjustment + movement: atomic with negative-stock protection.
