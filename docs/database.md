# Database Design V1

## Migration Plan

```text
000001_create_foundation.sql   IMPLEMENTED
000002_create_menu.sql         IMPLEMENTED
000003_create_inventory.sql    IMPLEMENTED
000004_procurement             NEXT
000005_receiving
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
- `id UUID PK`
- `code VARCHAR(50) UNIQUE NOT NULL`
- `name VARCHAR(100) NOT NULL`
- timestamps

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
- `id UUID PK`
- `code VARCHAR(30) UNIQUE NOT NULL`
- `name VARCHAR(100) NOT NULL`
- timestamps

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
- `created_by UUID nullable FK users`
- `updated_by UUID nullable FK users`
- timestamps

Indexes exist for `unit_id` and `is_active`.

## 000002 Menu

### periods
- `id UUID PK`
- `name VARCHAR(150)`
- `start_date DATE`
- `end_date DATE`
- `created_by UUID nullable FK users`
- timestamps

Database constraints:
- `end_date >= start_date`
- `end_date = start_date + 13`

This enforces one inclusive 14-day period.

### menu_templates
- `id UUID PK`
- `name VARCHAR(150)`
- `description TEXT nullable`
- `is_active BOOLEAN`
- `created_by`, `updated_by` nullable FK users
- timestamps

### menu_template_components
- `id UUID PK`
- `menu_template_id UUID FK menu_templates ON DELETE CASCADE`
- `name VARCHAR(150)`
- `sort_order INTEGER >= 0`
- timestamps

### menu_template_component_materials
- `id UUID PK`
- `menu_template_component_id UUID FK menu_template_components ON DELETE CASCADE`
- `material_id UUID FK materials`
- `qty_per_portion NUMERIC(18,4) > 0`
- `unit_id UUID FK units`
- timestamps

A material is unique inside one template component.

### scheduled_menus
- `id UUID PK`
- `period_id UUID FK periods`
- `menu_template_id UUID nullable source reference`
- `menu_date DATE`
- `planned_portions INTEGER > 0`
- `created_by`, `updated_by` nullable FK users
- timestamps

Constraint: one scheduled menu per period/date in V1.

### scheduled_menu_components
- Snapshot rows belonging to `scheduled_menus`.
- `source_template_component_id` is nullable and only preserves origin/audit linkage.
- Editing scheduled components does not mutate the source template.

### scheduled_menu_component_materials
- Snapshot material rows belonging to a scheduled component.
- `source_template_material_id` is nullable and only preserves origin/audit linkage.
- `qty_per_portion NUMERIC(18,4) > 0`.
- A material is unique inside one scheduled component.

## 000003 Inventory

### material_stocks
- `material_id UUID PK FK materials`
- `qty NUMERIC(18,4) NOT NULL DEFAULT 0`
- `unit_id UUID FK units`
- `updated_at TIMESTAMPTZ`

Constraint: `qty >= 0`.

`material_stocks` is the fast current-stock snapshot. The audit source of truth remains `stock_movements`.

### stock_movements
- `id UUID PK`
- `material_id UUID FK materials`
- `movement_type stock_movement_type`
- `reference_type stock_reference_type`
- `reference_id UUID`
- `qty NUMERIC(18,4) > 0`
- `unit_id UUID FK units`
- `movement_date TIMESTAMPTZ`
- `created_by UUID nullable FK users`
- `created_at TIMESTAMPTZ`

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

Indexes support material history and reference lookup.

### stock_reservations
- `id UUID PK`
- `scheduled_menu_id UUID FK scheduled_menus`
- `procurement_request_item_id UUID nullable`
- `material_id UUID FK materials`
- `qty NUMERIC(18,4) > 0`
- `unit_id UUID FK units`
- `status stock_reservation_status`
- `reserved_at`
- `released_at nullable`
- `consumed_at nullable`
- `created_by UUID nullable FK users`
- timestamps

Statuses:
- `ACTIVE`
- `CONSUMED`
- `RELEASED`

Lifecycle constraint:
- `ACTIVE`: neither released nor consumed timestamp exists.
- `RELEASED`: `released_at` exists and `consumed_at` is null.
- `CONSUMED`: `consumed_at` exists and `released_at` is null.

Reservation does not create a stock movement because physical stock has not changed.

`procurement_request_item_id` intentionally has no FK in migration `000003`; migration `000004_procurement` must add the FK after `procurement_request_items` exists.

## Planned 000004 Procurement

### procurement_requests
Planned statuses:
- `DRAFT`
- `WAITING_VERIFICATION`
- `VERIFIED`
- `REJECTED`

### procurement_request_items
Must preserve separately:
- `gross_requirement_qty`
- `existing_stock_qty`
- `reserved_stock_qty`
- `net_procurement_qty`

Formula:

```text
available_stock = current_stock - active_reserved_stock
net_procurement = max(gross_requirement - usable_available_stock, 0)
```

Procurement reservation must lock relevant inventory state so the same existing stock cannot be allocated to multiple future menu requirements.

### purchase_orders / purchase_order_items
Planned item statuses:
- `WAITING`
- `CANCELLED`
- `NOT_RECEIVED`
- `PARTIAL_RECEIVED`
- `RECEIVED`
- `OVER_RECEIVED`
- `FULFILLED`

Vendor is not master data; supplier/source information is stored on the transaction/item.

## Planned Receiving

Receipt rules:
- `excess_qty = max(received_qty - ordered_qty, 0)`
- `actual_amount = received_qty * agreed_unit_price`
- over-delivery is accepted in full and all accepted quantity enters stock.
- invoice/supporting documents are uploaded for Akuntan visibility; there is no payment workflow in the application.

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

Chef and Akuntan approvals are both required before stock OUT. Versioned approvals become invalid when the underlying submitted data is revised.

## Planned Stock Opname

Statuses:
- `DRAFT`
- `MATCHED`
- `DIFFERENCE_FOUND`
- `WAITING_ADJUSTMENT_APPROVAL`
- `COMPLETED`

`difference_qty = physical_qty - system_qty` is system-calculated. Differences do not alter stock automatically. Adjustment requires Chef + Akuntan approval.

## Important Transaction Rules

- Procurement reservation must lock relevant stock/reservation state before allocating existing stock.
- Receipt and stock IN must commit atomically.
- Direct purchase and stock IN must commit atomically.
- Approved usage and stock OUT must commit atomically.
- Stock OUT must fail when current stock is insufficient.
- Approved adjustment and stock movement must commit atomically.
- Old approvals remain historical but are invalid when `entity_version` no longer matches the current entity version.
