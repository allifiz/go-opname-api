# Requirements V1

This file is the repository-facing technical summary of the approved business flow. The approved meeting document (`rapat`) remains the source reference for the agreed business flow and is not modified by routine codebase work.

## Actors
- `CHEF`
- `AHLI_GIZI`
- `PENGAWAS_BAHAN_BAKU`
- `AKUNTAN`
- `KEPALA_SPPG`

`KEPALA_SPPG` exists as a role, but its operational permissions are still TBD.

## Period and Menu
- Ahli Gizi creates a period.
- One period lasts two weeks.
- Menu planning is performed per production day; one week normally has five production days and five scheduled menus.
- Menu is reusable as a template and is not permanently tied to one period.
- A scheduled menu is a snapshot/copy of the selected template so future template changes do not modify historical schedules.
- Scheduled menu components and materials may be customized independently from the source template.
- A menu is a package of components/condiments.
- Each component/condiment may contain multiple raw materials.
- Material requirement is stored per portion.
- Gross requirement is calculated as `qty_per_portion * planned_portions`.

## Procurement Stock Check
- Chef and Ahli Gizi determine production requirements without needing warehouse stock visibility.
- Their quantity is the immutable/auditable `gross_requirement`.
- Before Accountant verification, Pengawas Bahan Baku performs a procurement stock check.
- Procurement stock check considers usable available stock and active reservations.
- Existing stock used to offset procurement must not be counted twice across future scheduled menus.
- `available_stock = current_stock - active_reserved_stock`.
- `net_procurement = max(gross_requirement - available_stock, 0)`.
- Gross requirement, stock used/reserved, and net procurement remain separate values for audit.
- Accountant verifies the net procurement requirement, not the gross production requirement.

## Stock Reservation
- Existing stock allocated during procurement planning is reserved for the corresponding requirement/scheduled menu.
- Reservation does not reduce physical/current stock.
- Reservation prevents the same stock from being allocated to multiple future requirements.
- Reservation is consumed when approved usage uses the stock and released when the related requirement is cancelled or otherwise no longer needs it.

## Purchase Order
- PO is generated from verified net procurement.
- Vendor is not master data; supplier/source information is stored on the PO item or transaction.
- Price is agreed before PO and is treated as fixed for V1.
- Pengawas Bahan Baku may perform an H-1 stock re-check.
- An individual PO item may be cancelled at maximum H-1 when existing stock is sufficient.
- Cancellation reason must be stored for audit.
- Cancelling a PO item never changes historical gross requirement or the previously verified procurement calculation.

## Receiving
- Pengawas Bahan Baku records actual goods received.
- Wrong material delivery is rejected; V1 has no substitution workflow.
- Material quality rejection is outside V1 scope.
- Receipt statuses per PO item:
  - `NOT_RECEIVED`: received quantity = 0.
  - `PARTIAL_RECEIVED`: 0 < received quantity < ordered quantity.
  - `RECEIVED`: received quantity = ordered quantity.
  - `OVER_RECEIVED`: received quantity > ordered quantity.
  - `FULFILLED`: original need is fully satisfied through vendor receipt plus shortage purchase.
- Over-delivery is accepted in full.
- All actually received quantity enters stock.
- `excess_qty = max(received_qty - ordered_qty, 0)`.
- Actual vendor amount is based on actual received quantity and agreed unit price.
- `actual_amount = received_qty * agreed_unit_price`.
- Example: PO 20 kg total Rp500,000 means Rp25,000/kg. If 40 kg arrives, 40 kg is accepted, excess is 20 kg, and actual amount is Rp1,000,000.

## Direct Purchase
Two direct-purchase reasons exist:
- `SHORTAGE`: vendor supplied less than the PO requirement.
- `ADDITIONAL_REQUIREMENT`: production requirement increased after PO, such as increased portions.

Rules:
- `SHORTAGE` purchase must be linked to the deficient PO item.
- `SHORTAGE` quantity cannot exceed the remaining shortage.
- Additional portions after PO do not create a new PO; they use `ADDITIONAL_REQUIREMENT` direct purchase.
- Direct purchases store source/vendor text, quantity, unit, price, date, evidence/invoice, note, and actor.

## Invoice and Payment Scope
- Payment processing is not part of the application.
- Pengawas Bahan Baku uploads invoice/receipt/supporting documents.
- Accountant can view supporting documents and actual transaction values.

## Material Usage
- Pengawas Bahan Baku records actual material usage after production.
- Actual usage may differ from planning.
- Unused material remains stock.
- Material usage requires approval from both Chef and Accountant.
- Submission alone does not reduce stock.
- Stock OUT occurs only after both approvals are valid.
- If one approver rejects, status becomes `NEEDS_REVISION`.
- Any edit after approval invalidates the previous approvals and requires fresh approval.
- Approved stock OUT must be rejected when it would make stock negative.

## Stock Opname
- Pengawas Bahan Baku performs stock opname after production.
- System compares system stock and physical stock.
- `difference_qty = physical_qty - system_qty`.
- No difference means the opname can complete without adjustment.
- A difference does not automatically change stock.

## Stock Adjustment
- Opname difference creates an adjustment request.
- Adjustment requires both Chef and Accountant approval.
- Previous approval becomes invalid if the adjustment data is revised.
- Approved negative difference creates `ADJUSTMENT_OUT`.
- Approved positive difference creates `ADJUSTMENT_IN`.

## Inventory Audit
Every stock change must create a stock movement.

Minimum movement types:
- `IN`
- `OUT`
- `ADJUSTMENT_IN`
- `ADJUSTMENT_OUT`

Minimum references:
- `PO_RECEIPT`
- `SHORTAGE_PURCHASE`
- `ADDITIONAL_REQUIREMENT`
- `MATERIAL_USAGE`
- `STOCK_ADJUSTMENT`

`stock_movements` is the inventory audit source of truth. `material_stocks` may be maintained as a read-optimized current snapshot.

## Units V1
- `KG`
- `PCS`
- `LT`
- `IKAT`
- `RENCENG`
- `BOTOL`

## Explicitly Out of Scope V1
- FIFO.
- Per-batch inventory tracking.
- Vendor master.
- Payment execution/workflow.
- Product-quality rejection workflow.
- Material substitution workflow.
- Negative stock.
