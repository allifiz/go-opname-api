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
000008_material_usage.sql                NEXT
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
Foundation, reusable menu templates, scheduled-menu snapshots, stock ledger/reservations, procurement requests, PO generation, H-1 cancellation, and receiving are implemented.

Procurement allocation:
```text
active_reserved_stock = SUM(ACTIVE reservation qty)
available_stock = MAX(current_stock - active_reserved_stock, 0)
allocation_qty = MIN(gross_requirement, available_stock)
net_procurement = MAX(gross_requirement - available_stock, 0)
```

## Receiving
Multiple receipts per PO item are supported. Cumulative vendor receipt quantity drives `NOT_RECEIVED`, `PARTIAL_RECEIVED`, `RECEIVED`, and `OVER_RECEIVED`. Positive receipts atomically create stock `IN` + `PO_RECEIPT` movement. Zero receipt quantity is a valid operational event and creates no movement.

## 000007 Direct Purchase

### direct_purchase_type
- `SHORTAGE`
- `ADDITIONAL_REQUIREMENT`

### additional_requirements
Records a production portion increase without mutating the original procurement snapshot.

Fields:
- `scheduled_menu_id`
- `previous_portions > 0`
- `new_portions > previous_portions`
- `created_by`
- `created_at`

The scheduled-menu row is locked while current effective portions are resolved. Current effective portions are the most recent `additional_requirements.new_portions`, falling back to original `scheduled_menus.planned_portions`.

### additional_requirement_items
Server-calculated material requirements for the portion delta.

```text
additional_portions = new_portions - current_effective_portions
additional_qty(material) = SUM(snapshot qty_per_portion * additional_portions)
```

A material appears once per additional requirement after aggregation across scheduled-menu components.

### direct_purchases
Header fields:
- `scheduled_menu_id`
- `purchase_type`
- `source_name`
- `purchase_date`
- `purchased_by`
- `note`

Vendor/source remains transaction data, not master data.

### direct_purchase_items
Each item belongs to exactly one business source:
- a `purchase_order_item_id` for `SHORTAGE`; or
- an `additional_requirement_item_id` for `ADDITIONAL_REQUIREMENT`.

Constraint requires exactly one of those references to be non-null.

Fields include:
- `material_id`
- `qty > 0`
- `unit_id`
- `unit_price >= 0`
- generated `total_amount = ROUND(qty * unit_price, 2)`

## SHORTAGE Calculation and Transaction
PO item is locked before calculating shortage.

```text
remaining_shortage_qty = MAX(
    ordered_qty
    - SUM(receipt_items.received_qty)
    - SUM(SHORTAGE direct_purchase_items.qty),
    0
)
```

Rules:
- shortage purchase qty must be `> 0`;
- shortage purchase qty cannot exceed `remaining_shortage_qty`;
- material/unit come from the locked PO item;
- direct purchase item + stock increase + `SHORTAGE_PURCHASE` movement commit atomically;
- if purchase qty exactly equals remaining shortage, PO item becomes `FULFILLED` and PO header is recalculated.

## ADDITIONAL_REQUIREMENT Transaction
The service executes atomically:

```text
lock scheduled menu
resolve current effective portions
require new_portions > current effective portions
calculate additional material qty from scheduled snapshot
require exactly one client price per calculated material
create additional_requirements header
create additional_requirement_items
create ADDITIONAL_REQUIREMENT direct purchase
for each material:
    create direct_purchase_item
    ensure + lock material stock
    increase stock
    create stock movement IN / ADDITIONAL_REQUIREMENT
commit
```

Original procurement gross/existing/reserved/net snapshots remain unchanged.

## sqlc Query Files
Implemented:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`
- `internal/database/query/receiving.sql`
- `internal/database/query/direct_purchase.sql`

## Planned Usage / Opname
Material usage and stock adjustment both require Chef + Accountant approval. Stock changes occur only after valid approvals for the current entity version.

## Important Transaction Rules
- Procurement stock check + reservation: atomic with stock row lock.
- H-1 replacement reservation + PO-item cancellation: atomic.
- Receipt + stock IN: atomic with PO-item lock.
- SHORTAGE purchase + stock IN: atomic with PO-item lock and remaining-shortage calculation.
- ADDITIONAL_REQUIREMENT + stock IN: atomic with scheduled-menu lock.
- Approved usage + stock OUT: atomic.
- Approved adjustment + stock movement: atomic.
