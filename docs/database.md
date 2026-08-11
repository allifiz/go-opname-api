# Database Design V1

## Migration Plan

```text
000001_create_foundation.sql   IMPLEMENTED
000002_create_menu.sql         IMPLEMENTED
000003_create_inventory.sql    IMPLEMENTED
000004_create_procurement.sql  IMPLEMENTED
000005_receiving               NEXT
000006_direct_purchase
000007_material_usage
000008_stock_opname
```

> The starter `000001` migration was intentionally rewritten because this project is still in the initial build phase. A local database that previously applied the old starter migration must be recreated/reset before applying the new schema.

## General Conventions
- Primary keys use `UUID DEFAULT gen_random_uuid()`.
- Quantity fields use `NUMERIC(18,4)`.
- Money fields use `NUMERIC(18,2)`.
- Operational timestamps use `TIMESTAMPTZ`.
- Production/menu calendar dates use `DATE`.
- Foreign keys default to `RESTRICT` for transactional/master data that must not disappear from history.
- User audit references may use `ON DELETE SET NULL` so historical business records survive user removal/deactivation.
- Negative stock is forbidden.

## 000001 Foundation

### roles
Seed codes:
- `CHEF`
- `AHLI_GIZI`
- `PENGAWAS_BAHAN_BAKU`
- `AKUNTAN`
- `KEPALA_SPPG`

### users
- `id UUID PK`
- `role_id UUID FK roles`
- `name VARCHAR(150)`
- `email VARCHAR(150) UNIQUE`
- `password_hash TEXT`
- `is_active BOOLEAN`
- timestamps

### units
Seed codes:
- `KG`
- `PCS`
- `LT`
- `IKAT`
- `RENCENG`
- `BOTOL`

### materials
- `id UUID PK`
- `name VARCHAR(150) UNIQUE NOT NULL`
- `unit_id UUID FK units`
- `is_active BOOLEAN`
- `created_by`, `updated_by` nullable FK users
- timestamps

## 000002 Menu

### periods
Database constraints:
- `end_date >= start_date`
- `end_date = start_date + 13`

This enforces one inclusive 14-day period.

### menu_templates
Reusable menu library records. A template is not permanently tied to a period.

### menu_template_components
Template condiment/component rows with `sort_order >= 0`.

### menu_template_component_materials
- material per template component
- `qty_per_portion NUMERIC(18,4) > 0`
- material is unique inside one component

### scheduled_menus
- belongs to a period
- optional `menu_template_id` origin reference
- `menu_date DATE`
- `planned_portions > 0`
- one scheduled menu per period/date in V1

### scheduled_menu_components
Snapshot rows copied from a template and independently editable.

### scheduled_menu_component_materials
Snapshot material rows. `source_template_material_id` only preserves origin linkage and does not make history depend on the current template.

## 000003 Inventory

### material_stocks
Fast current-stock snapshot.

Constraint:
- `qty >= 0`

The audit source of truth remains `stock_movements`.

### stock_movements
Movement enum:
- `IN`
- `OUT`
- `ADJUSTMENT_IN`
- `ADJUSTMENT_OUT`

Reference enum:
- `PO_RECEIPT`
- `SHORTAGE_PURCHASE`
- `ADDITIONAL_REQUIREMENT`
- `MATERIAL_USAGE`
- `STOCK_ADJUSTMENT`

`qty` must be positive. Indexes support material history and reference lookup.

### stock_reservations
Statuses:
- `ACTIVE`
- `CONSUMED`
- `RELEASED`

Lifecycle constraint:
- `ACTIVE`: neither release nor consume timestamp exists.
- `RELEASED`: `released_at` exists and `consumed_at` is null.
- `CONSUMED`: `consumed_at` exists and `released_at` is null.

Reservation is an allocation only and does not create a stock movement.

`procurement_request_item_id` is created nullable in `000003`. The FK is attached by `000004` after `procurement_request_items` exists.

## 000004 Procurement

### procurement_requests
Statuses:
- `DRAFT`
- `WAITING_VERIFICATION`
- `VERIFIED`
- `REJECTED`

Fields preserve stock-check and verification audit timestamps/users. A `VERIFIED` row requires both `verified_by` and `verified_at`.

### procurement_request_items
Preserved separately:
- `gross_requirement_qty`
- `existing_stock_qty`
- `reserved_stock_qty`
- `net_procurement_qty`

All quantity values must be non-negative and `net_procurement_qty <= gross_requirement_qty`.

A material may appear only once in one procurement request.

Business calculation performed transactionally by the service layer:

```text
active_reserved_stock = sum(ACTIVE reservations for material)
available_stock = max(current_stock - active_reserved_stock, 0)
usable_existing_stock = min(gross_requirement, available_stock)
net_procurement = max(gross_requirement - usable_existing_stock, 0)
```

The stock row/reservation state must be locked while calculating and allocating to prevent the same physical stock from offsetting multiple future requirements.

### deferred reservation FK
`stock_reservations.procurement_request_item_id` now references `procurement_request_items.id` with `ON DELETE RESTRICT`.

### purchase_orders
Header statuses:
- `DRAFT`
- `VERIFIED`
- `PARTIALLY_RECEIVED`
- `COMPLETED`

Important fields:
- `procurement_request_id`
- `scheduled_menu_id`
- unique `po_number`
- `delivery_date`
- `created_by`

### purchase_order_items
Item statuses:
- `WAITING`
- `CANCELLED`
- `NOT_RECEIVED`
- `PARTIAL_RECEIVED`
- `RECEIVED`
- `OVER_RECEIVED`
- `FULFILLED`

Important fields:
- `procurement_request_item_id`
- `material_id`
- `ordered_qty > 0`
- `agreed_unit_price >= 0`
- `supplier_name`
- cancellation audit fields

Vendor is intentionally not master data. Supplier name is stored per PO item.

Cancellation lifecycle constraint requires `cancelled_at`, `cancelled_by`, and `cancel_reason` when status becomes `CANCELLED`.

H-1 timing is a cross-table/date business rule and is enforced transactionally in the service layer before an item is cancelled.

## sqlc Query Files
Implemented source queries:
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`

They cover the initial master/material/menu/scheduled-menu/inventory/reservation/procurement/PO operations needed by the service layer.

`sqlc generate` still requires local validation after migrations are applied to a clean development setup.

## Planned Receiving
Receipt rules:
- `excess_qty = max(received_qty - ordered_qty, 0)`
- `actual_amount = received_qty * agreed_unit_price`
- over-delivery is accepted in full and all actual received quantity enters stock
- invoice/supporting documents are uploaded for Akuntan visibility
- there is no payment workflow in the application

## Planned Direct Purchase
Types:
- `SHORTAGE`
- `ADDITIONAL_REQUIREMENT`

Rules:
- `SHORTAGE` cannot exceed remaining shortage.
- Additional portions after PO create `ADDITIONAL_REQUIREMENT`, not a second PO.

## Planned Material Usage
Statuses:
- `DRAFT`
- `WAITING_APPROVAL`
- `APPROVED`
- `NEEDS_REVISION`

Chef and Akuntan approvals are both required before stock OUT. Versioned approvals become invalid when submitted data is revised.

## Planned Stock Opname
Statuses:
- `DRAFT`
- `MATCHED`
- `DIFFERENCE_FOUND`
- `WAITING_ADJUSTMENT_APPROVAL`
- `COMPLETED`

`difference_qty = physical_qty - system_qty` is system-calculated. Differences do not alter stock automatically. Adjustment requires Chef + Akuntan approval.

## Important Transaction Rules
- Procurement stock check + reservation must commit atomically and lock relevant inventory/reservation state.
- PO item H-1 cancellation must be validated against the PO delivery date in the same business operation.
- Receipt and stock IN must commit atomically.
- Direct purchase and stock IN must commit atomically.
- Approved usage and stock OUT must commit atomically.
- Stock OUT must fail when current stock is insufficient.
- Approved adjustment and stock movement must commit atomically.
- Old approvals remain historical but are invalid when `entity_version` no longer matches the current entity version.
