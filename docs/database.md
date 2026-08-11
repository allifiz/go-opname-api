# Database Design V1

## Migration Plan

```text
000001_foundation
000002_menu
000003_inventory
000004_procurement
000005_receiving
000006_direct_purchase
000007_material_usage
000008_stock_opname
```

## Foundation

### roles
- `id UUID PK`
- `code VARCHAR UNIQUE NOT NULL`
- `name VARCHAR NOT NULL`

Seed codes:
- `CHEF`
- `AHLI_GIZI`
- `PENGAWAS_BAHAN_BAKU`
- `AKUNTAN`
- `KEPALA_SPPG`

### users
- `id UUID PK`
- `role_id UUID FK roles`
- `name`
- `email UNIQUE`
- `password_hash`
- `is_active`
- timestamps

### units
- `id UUID PK`
- `code VARCHAR UNIQUE NOT NULL`
- `name VARCHAR NOT NULL`

Seed codes:
- `KG`
- `PCS`
- `LT`
- `IKAT`
- `RENCENG`
- `BOTOL`

### materials
- `id UUID PK`
- `name VARCHAR UNIQUE NOT NULL`
- `unit_id UUID FK units`
- `is_active BOOLEAN`
- audit timestamps

## Menu

### periods
- `id`
- `name`
- `start_date DATE`
- `end_date DATE`
- `created_by FK users`
- timestamps

Business rule: one period lasts two weeks.

### menu_templates
- `id`
- `name`
- `description`
- `created_by`
- `is_active`
- timestamps

### menu_template_components
- `id`
- `menu_template_id`
- `name`
- `sort_order`
- timestamps

### menu_template_component_materials
- `id`
- `menu_template_component_id`
- `material_id`
- `qty_per_portion NUMERIC(18,4)`
- `unit_id`
- timestamps

### scheduled_menus
- `id`
- `period_id`
- `menu_template_id` nullable source reference
- `menu_date DATE`
- `planned_portions INTEGER`
- `created_by`
- timestamps

### scheduled_menu_components
- `id`
- `scheduled_menu_id`
- `source_template_component_id` nullable
- `name`
- `sort_order`
- timestamps

### scheduled_menu_component_materials
- `id`
- `scheduled_menu_component_id`
- `source_template_material_id` nullable
- `material_id`
- `qty_per_portion NUMERIC(18,4)`
- `unit_id`
- timestamps

## Inventory

### material_stocks
- `material_id UUID PK FK materials`
- `qty NUMERIC(18,4) NOT NULL DEFAULT 0`
- `unit_id UUID FK units`
- `updated_at`

Constraint: quantity cannot be negative.

### stock_movements
- `id`
- `material_id`
- `movement_type`
- `reference_type`
- `reference_id UUID`
- `qty NUMERIC(18,4)`
- `unit_id`
- `movement_date TIMESTAMPTZ`
- `created_by`
- `created_at`

Movement types:
- `IN`
- `OUT`
- `ADJUSTMENT_IN`
- `ADJUSTMENT_OUT`

Reference types:
- `PO_RECEIPT`
- `SHORTAGE_PURCHASE`
- `ADDITIONAL_REQUIREMENT`
- `MATERIAL_USAGE`
- `STOCK_ADJUSTMENT`

### stock_reservations
- `id`
- `scheduled_menu_id`
- `procurement_request_item_id` nullable until procurement schema exists
- `material_id`
- `qty NUMERIC(18,4)`
- `unit_id`
- `status`
- `reserved_at`
- `released_at` nullable
- `consumed_at` nullable
- `created_by`
- timestamps

Statuses:
- `ACTIVE`
- `CONSUMED`
- `RELEASED`

Reservation does not create a stock movement.

## Procurement

### procurement_requests
- `id`
- `scheduled_menu_id`
- `status`
- `checked_by`
- `checked_at`
- `submitted_at`
- `verified_by`
- `verified_at`
- timestamps

Statuses:
- `DRAFT`
- `WAITING_VERIFICATION`
- `VERIFIED`
- `REJECTED`

### procurement_request_items
- `id`
- `procurement_request_id`
- `material_id`
- `gross_requirement_qty NUMERIC(18,4)`
- `existing_stock_qty NUMERIC(18,4)`
- `reserved_stock_qty NUMERIC(18,4)`
- `net_procurement_qty NUMERIC(18,4)`
- `unit_id`
- timestamps

### purchase_orders
- `id`
- `procurement_request_id`
- `scheduled_menu_id`
- `po_number`
- `delivery_date DATE`
- `status`
- `created_by`
- timestamps

### purchase_order_items
- `id`
- `purchase_order_id`
- `procurement_request_item_id`
- `material_id`
- `ordered_qty NUMERIC(18,4)`
- `unit_id`
- `agreed_unit_price NUMERIC(18,2)`
- `supplier_name`
- `status`
- `cancelled_at`
- `cancelled_by`
- `cancel_reason`
- timestamps

Item statuses:
- `WAITING`
- `CANCELLED`
- `NOT_RECEIVED`
- `PARTIAL_RECEIVED`
- `RECEIVED`
- `OVER_RECEIVED`
- `FULFILLED`

## Receiving

### receipts
- `id`
- `purchase_order_id`
- `received_at`
- `received_by`
- `note`
- timestamps

### receipt_items
- `id`
- `receipt_id`
- `purchase_order_item_id`
- `material_id`
- `ordered_qty`
- `received_qty`
- `excess_qty`
- `unit_id`
- `agreed_unit_price`
- `actual_amount`
- timestamps

Rules:
- `excess_qty = max(received_qty - ordered_qty, 0)`
- `actual_amount = received_qty * agreed_unit_price`

### receipt_documents
- `id`
- `receipt_id`
- `document_type`
- `file_url`
- `file_name`
- `uploaded_by`
- `created_at`

## Direct Purchase

### additional_requirements
- `id`
- `scheduled_menu_id`
- `previous_portions`
- `new_portions`
- `created_by`
- `created_at`

### additional_requirement_items
- `id`
- `additional_requirement_id`
- `material_id`
- `additional_qty`
- `unit_id`
- `created_at`

### direct_purchases
- `id`
- `scheduled_menu_id`
- `purchase_type`
- `source_name`
- `purchase_date`
- `purchased_by`
- `note`
- timestamps

Purchase types:
- `SHORTAGE`
- `ADDITIONAL_REQUIREMENT`

### direct_purchase_items
- `id`
- `direct_purchase_id`
- `purchase_order_item_id` nullable
- `additional_requirement_item_id` nullable
- `material_id`
- `qty`
- `unit_id`
- `unit_price`
- `total_amount`
- timestamps

Constraint: `SHORTAGE` quantity cannot exceed the remaining shortage.

## Material Usage

### material_usages
- `id`
- `scheduled_menu_id`
- `usage_date`
- `submitted_by`
- `status`
- `version INTEGER DEFAULT 1`
- `submitted_at`
- timestamps

Statuses:
- `DRAFT`
- `WAITING_APPROVAL`
- `APPROVED`
- `NEEDS_REVISION`

### material_usage_items
- `id`
- `material_usage_id`
- `material_id`
- `planned_qty`
- `actual_qty`
- `unit_id`
- timestamps

### material_usage_approvals
- `id`
- `material_usage_id`
- `approver_role`
- `approver_id`
- `entity_version`
- `status`
- `note`
- `approved_at`
- timestamps

## Stock Opname

### stock_opnames
- `id`
- `scheduled_menu_id`
- `opname_date`
- `performed_by`
- `status`
- timestamps

Statuses:
- `DRAFT`
- `MATCHED`
- `DIFFERENCE_FOUND`
- `WAITING_ADJUSTMENT_APPROVAL`
- `COMPLETED`

### stock_opname_items
- `id`
- `stock_opname_id`
- `material_id`
- `system_qty`
- `physical_qty`
- `difference_qty`
- `unit_id`
- timestamps

Rule: `difference_qty = physical_qty - system_qty` and is system-calculated.

### stock_adjustments
- `id`
- `stock_opname_item_id`
- `material_id`
- `adjustment_qty`
- `reason`
- `submitted_by`
- `status`
- `version INTEGER DEFAULT 1`
- timestamps

### stock_adjustment_approvals
- `id`
- `stock_adjustment_id`
- `approver_role`
- `approver_id`
- `entity_version`
- `status`
- `note`
- `approved_at`
- timestamps

## Important Transaction Rules
- Procurement reservation must lock relevant stock/reservation state before allocating existing stock.
- Receipt and stock IN must commit atomically.
- Direct purchase and stock IN must commit atomically.
- Approved usage and stock OUT must commit atomically.
- Stock OUT must fail when current stock is insufficient.
- Approved adjustment and stock movement must commit atomically.
- Old approvals remain historical but are invalid when `entity_version` no longer matches the current entity version.
