# Database Design V1

## Migration Plan

```text
000001_create_foundation.sql             IMPLEMENTED
000002_create_menu.sql                   IMPLEMENTED
000003_create_inventory.sql              IMPLEMENTED
000004_create_procurement.sql            IMPLEMENTED
000005_purchase_order_constraints.sql    IMPLEMENTED
000006_receiving.sql                     NEXT
000007_direct_purchase.sql
000008_material_usage.sql
000009_stock_opname.sql
```

> The starter `000001` migration was intentionally rewritten because this project is still in the initial build phase. A local database that previously applied the old starter migration must be recreated/reset before applying the current schema.

## General Conventions
- Primary keys: `UUID DEFAULT gen_random_uuid()`.
- Quantities: `NUMERIC(18,4)`.
- Money: `NUMERIC(18,2)`.
- Operational timestamps: `TIMESTAMPTZ`.
- Production/menu dates: `DATE`.
- Transaction/master history defaults to restrictive foreign keys.
- User audit references may use `ON DELETE SET NULL`.
- Negative stock is forbidden.

## Foundation
`roles`, `users`, `units`, and `materials` are implemented. Approved role seeds are `CHEF`, `AHLI_GIZI`, `PENGAWAS_BAHAN_BAKU`, `AKUNTAN`, and `KEPALA_SPPG`. Approved unit seeds are `KG`, `PCS`, `LT`, `IKAT`, `RENCENG`, and `BOTOL`.

## Menu
A period is exactly 14 inclusive days. Menu templates are reusable. Scheduled menus copy template components/materials into snapshot tables so historical production requirements do not depend on later template edits.

Gross requirement is calculated from scheduled snapshots:

```text
gross_requirement(material) = SUM(qty_per_portion * planned_portions)
```

## Inventory

### material_stocks
Current read-optimized stock snapshot. `qty >= 0`.

### stock_movements
Audit ledger for physical stock changes. Reservations never create stock movements.

### stock_reservations
Statuses: `ACTIVE`, `CONSUMED`, `RELEASED`.

Procurement allocation uses:

```text
active_reserved_stock = SUM(ACTIVE reservation qty)
available_stock = MAX(current_stock - active_reserved_stock, 0)
allocation_qty = MIN(gross_requirement, available_stock)
net_procurement = MAX(gross_requirement - available_stock, 0)
```

The `material_stocks` row is locked `FOR UPDATE` before availability is calculated. This serializes allocations for the same material.

For H-1 cancellation, the service also calculates currently-unreserved stock as:

```text
unreserved_stock = MAX(current_stock - SUM(all ACTIVE reservations), 0)
```

The material stock row stays locked while the additional reservation and PO-item cancellation are performed.

## Procurement

### procurement_requests
Statuses: `DRAFT`, `WAITING_VERIFICATION`, `VERIFIED`, `REJECTED`.

`scheduled_menu_id` is unique, so one scheduled menu has one procurement-request lifecycle.

### procurement_request_items
Persist separately:
- `gross_requirement_qty`
- `existing_stock_qty`
- `reserved_stock_qty`
- `net_procurement_qty`

A material may appear once per procurement request.

### purchase_orders
Statuses: `DRAFT`, `VERIFIED`, `PARTIALLY_RECEIVED`, `COMPLETED`.

Important fields:
- `procurement_request_id`
- `scheduled_menu_id`
- unique `po_number`
- `delivery_date`
- `created_by`

`000005_purchase_order_constraints.sql` adds a unique constraint on `procurement_request_id`, enforcing one PO per procurement request in V1.

Generated POs are created in `VERIFIED` status after the procurement request itself is `VERIFIED`.

### purchase_order_items
Statuses:
- `WAITING`
- `CANCELLED`
- `NOT_RECEIVED`
- `PARTIAL_RECEIVED`
- `RECEIVED`
- `OVER_RECEIVED`
- `FULFILLED`

`ordered_qty` is not client-controlled. It is copied from the verified procurement item's positive `net_procurement_qty`.

Supplier is intentionally stored per item, not as vendor master data. `agreed_unit_price` is fixed for the PO item.

Cancellation lifecycle requires `cancelled_at`, `cancelled_by`, and `cancel_reason` whenever status is `CANCELLED`.

## H-1 Cancellation Transaction
The service executes the following atomically:

```text
lock PO item + parent PO
validate item status = WAITING
validate business date < delivery_date
lock material_stocks row
calculate currently-unreserved stock
require unreserved_stock >= ordered_qty
create ACTIVE reservation for ordered_qty
cancel PO item with reason EXISTING_STOCK_SUFFICIENT
commit
```

If any step fails, both reservation and cancellation roll back.

The extra reservation is linked to the same `procurement_request_item_id`. Combined with the original procurement reservation, it protects the full scheduled-menu requirement after the purchased quantity is cancelled.

## sqlc Query Files
Implemented:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`

The query layer covers menu gross aggregation, stock locking/availability, reservations, procurement lifecycle, PO generation/read operations, and PO-item locking for H-1 cancellation.

`sqlc generate`, migrations, rollback, tests, and build are validated by GitHub Actions against PostgreSQL 17.

## Planned Receiving
Receiving now starts at `000006_receiving.sql` because migration `000005` is used for PO constraints.

Required rules:
- multiple receipts per PO item are supported;
- cumulative vendor receipt quantity drives item status;
- `initial_shortage_qty = MAX(ordered_qty - total_vendor_received_qty, 0)`;
- over-delivery is based on cumulative received quantity;
- all accepted actual quantity enters stock;
- actual amount uses fixed `agreed_unit_price`;
- receipt + stock IN must commit atomically;
- invoice/supporting document metadata is stored for Accountant visibility;
- no payment workflow is implemented.

## Planned Direct Purchase
Types: `SHORTAGE`, `ADDITIONAL_REQUIREMENT`.

`SHORTAGE` cannot exceed remaining shortage. Additional production need after PO uses `ADDITIONAL_REQUIREMENT`, not another PO.

## Planned Usage / Opname
Material usage and stock adjustment both require Chef + Accountant approval. Actual stock changes happen only after valid approvals for the current entity version. Negative stock remains forbidden.

## Important Transaction Rules
- Procurement stock check + reservation: atomic with stock row lock.
- H-1 replacement reservation + PO-item cancellation: atomic.
- Receipt + stock IN: atomic.
- Direct purchase + stock IN: atomic.
- Approved usage + stock OUT: atomic.
- Approved adjustment + stock movement: atomic.
