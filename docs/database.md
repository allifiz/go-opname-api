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
000009_stock_opname.sql                  NEXT
```

## General Conventions
- Primary keys: `UUID DEFAULT gen_random_uuid()`.
- Quantities: `NUMERIC(18,4)`.
- Money: `NUMERIC(18,2)`.
- Operational timestamps: `TIMESTAMPTZ`.
- Production/menu dates: `DATE`.
- Negative stock is forbidden.
- Real inventory changes must be represented in `stock_movements`.

## Foundation / Menu / Procurement / Receiving / Direct Purchase
Foundation, reusable menu templates, scheduled-menu snapshots, stock reservations, procurement requests, PO generation, H-1 cancellation, cumulative receiving, and direct purchase are implemented.

Procurement allocation remains:
```text
active_reserved_stock = SUM(ACTIVE reservation qty)
available_stock = MAX(current_stock - active_reserved_stock, 0)
allocation_qty = MIN(gross_requirement, available_stock)
net_procurement = MAX(gross_requirement - available_stock, 0)
```

Receiving supports cumulative receipt quantities. Direct purchase supports `SHORTAGE` and `ADDITIONAL_REQUIREMENT`, both applying stock `IN` atomically with ledger movements.

## 000008 Material Usage

### material_usage_status
- `DRAFT`
- `WAITING_APPROVAL`
- `APPROVED`
- `NEEDS_REVISION`

### material_usages
One material-usage lifecycle exists per scheduled menu.

Important fields:
- `scheduled_menu_id UNIQUE`
- `usage_date`
- `submitted_by`
- `status`
- `version > 0`
- `submitted_at`
- `applied_at`
- timestamps

`version` is monotonically incremented whenever editable usage data is revised. Historical approvals are never deleted.

### material_usage_items
Fields:
- `material_usage_id`
- `material_id`
- `planned_qty >= 0`
- `actual_qty >= 0`
- `unit_id`

A material is unique per usage.

`planned_qty` is server-derived from the scheduled-menu snapshot and the latest effective portion count:

```text
effective_portions = latest additional_requirements.new_portions
                     OR scheduled_menus.planned_portions
planned_qty(material) = SUM(snapshot qty_per_portion * effective_portions)
```

The client controls only `actual_qty`, not `planned_qty`.

### material_usage_approvals
Approver roles:
- `CHEF`
- `AKUNTAN`

Decisions:
- `APPROVED`
- `REJECTED`

Important fields:
- `material_usage_id`
- `approver_role`
- `approver_id`
- `entity_version`
- `status`
- `note`
- `decided_at`

Unique constraint:
```text
(material_usage_id, approver_role, entity_version)
```

This guarantees at most one Chef decision and one Accountant decision for one usage version.

## Revision / Approval Semantics

Editable states:
- `DRAFT`
- `NEEDS_REVISION`

A revision:
1. locks the usage row;
2. increments `version`;
3. resets status to `DRAFT`;
4. clears submission/application timestamps;
5. replaces current usage-item rows;
6. preserves all previous approval rows as immutable history.

Only approvals whose `entity_version` equals the current usage `version` count toward final approval.

A rejection while `WAITING_APPROVAL` changes status to `NEEDS_REVISION` and does not change stock.

## Dual Approval + Stock OUT Transaction

The second valid approval for the current version triggers application in the same transaction:

```text
lock material_usage
validate WAITING_APPROVAL
validate approver is active CHEF or AKUNTAN
insert versioned approval
count current-version approved roles
if only one role approved:
    commit approval only
if both roles approved:
    for each usage item:
        ensure material_stocks row
        lock material_stocks row
        require stock >= actual_qty
        decrement stock
        create OUT / MATERIAL_USAGE stock movement
        consume ACTIVE reservations for scheduled menu + material
    mark material_usage APPROVED + applied_at
commit
```

If any material lacks sufficient stock, the entire second-approval transaction rolls back. This means the second approval row, all stock decrements, all movements, reservation consumption, and final `APPROVED` status are all rejected together.

Zero actual usage creates no stock movement but still consumes the associated active reservation because the scheduled production allocation is complete for that material.

## Inventory Queries Added for Usage
- `DecreaseMaterialStockIfSufficient`
- `ConsumeActiveReservationsByScheduledMenuMaterial`

`DecreaseMaterialStockIfSufficient` includes `qty >= subtract_qty` in the SQL predicate, so negative stock cannot be created even under concurrent operations.

## sqlc Query Files
Implemented:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`
- `internal/database/query/receiving.sql`
- `internal/database/query/direct_purchase.sql`
- `internal/database/query/material_usage.sql`

## Next: Stock Opname
Planned `000009_stock_opname.sql`:
- stock-opname header/items;
- system vs physical quantity snapshot;
- server-calculated difference;
- no automatic stock correction;
- versioned Chef + Accountant approval for adjustment;
- approved adjustment creates `ADJUSTMENT_IN` or `ADJUSTMENT_OUT` movement atomically.

## Important Transaction Rules
- Procurement stock check + reservation: atomic with stock row lock.
- H-1 replacement reservation + PO-item cancellation: atomic.
- Receipt + stock IN: atomic with PO-item lock.
- SHORTAGE purchase + stock IN: atomic with PO-item lock.
- ADDITIONAL_REQUIREMENT + stock IN: atomic with scheduled-menu lock.
- Dual-approved material usage + stock OUT + reservation consumption: atomic.
- Approved stock adjustment + movement: planned atomic operation.
