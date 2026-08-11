# API Plan V1

This file tracks the HTTP contract surface. Detailed schemas belong in code/OpenAPI once implemented.

## Conventions
- Base path: `/api/v1`.
- JSON request/response.
- Validation errors -> `400`.
- Missing rows -> `404`.
- Conflict/invalid transition -> `409`.
- Business quantity/stock violations -> `422`.
- Unexpected failures -> `500`.
- Authentication is not implemented yet; audit user IDs are temporarily supplied in request bodies where required.
- Quantities and prices are represented as JSON strings when mapped to PostgreSQL `NUMERIC`.

## Master / Menu
Implemented:
- `GET /health`
- `GET /api/v1/units`
- `GET /api/v1/materials`
- `GET /api/v1/materials/:id`
- `POST /api/v1/materials`
- `PUT /api/v1/materials/:id`
- `GET /api/v1/periods`
- `POST /api/v1/periods`
- `GET /api/v1/menu-templates`
- `GET /api/v1/menu-templates/:id`
- `POST /api/v1/menu-templates`
- `POST /api/v1/scheduled-menus`
- `GET /api/v1/scheduled-menus/:id`

Menu-template update remains planned.

## Procurement / PO
Implemented:
- `POST /api/v1/scheduled-menus/:id/procurement-stock-check`
- `GET /api/v1/scheduled-menus/:id/procurement-requests`
- `GET /api/v1/procurement-requests/:id`
- `POST /api/v1/procurement-requests/:id/submit`
- `POST /api/v1/procurement-requests/:id/verify`
- `POST /api/v1/procurement-requests/:id/reject`
- `POST /api/v1/procurement-requests/:id/purchase-order`
- `GET /api/v1/purchase-orders/:id`
- `GET /api/v1/scheduled-menus/:id/purchase-orders`
- `POST /api/v1/purchase-order-items/:id/cancel-h1`

PO ordered quantity is server-controlled from verified `net_procurement_qty`. H-1 cancellation creates replacement reservation atomically before cancelling the item.

## Receiving
Implemented:
- `POST /api/v1/purchase-orders/:id/receipts`
- `GET /api/v1/purchase-orders/:id/receipts`
- `GET /api/v1/receipts/:id`

Receiving supports multiple receipts per PO item and cumulative status:
- cumulative = 0 -> `NOT_RECEIVED`
- cumulative < ordered -> `PARTIAL_RECEIVED`
- cumulative = ordered -> `RECEIVED`
- cumulative > ordered -> `OVER_RECEIVED`

Positive receipt quantity creates stock `IN` + `PO_RECEIPT` movement atomically. Zero quantity is valid and creates no movement. Receipt document metadata supports `INVOICE`, `NOTA`, `PHOTO`, `OTHER`.

## Direct Purchase

### SHORTAGE
- `POST /api/v1/purchase-order-items/:id/direct-purchases/shortage`
- Status: implemented, latest CI validation in progress.

Example body:
```json
{
  "qty": "2.5000",
  "unit_price": "31000.00",
  "source_name": "Pasar Lokal",
  "purchased_by": "<existing-user-uuid>",
  "note": "Menutup kekurangan vendor"
}
```

Server calculates:
```text
remaining_shortage = max(
  ordered_qty
  - cumulative_vendor_received_qty
  - cumulative_SHORTAGE_direct_purchase_qty,
  0
)
```

Rules:
- PO item is locked before shortage calculation.
- `qty` must be positive and cannot exceed `remaining_shortage`.
- Excess returns `422` (`SHORTAGE_QTY_EXCEEDED`).
- Material/unit come from the PO item.
- Purchase creates stock `IN` + `SHORTAGE_PURCHASE` movement atomically.
- When the purchase exactly closes remaining shortage, PO item becomes `FULFILLED` and PO header status is recalculated.

### ADDITIONAL_REQUIREMENT
- `POST /api/v1/scheduled-menus/:id/direct-purchases/additional-requirement`
- Status: implemented, latest CI validation in progress.

Example body:
```json
{
  "new_portions": 550,
  "source_name": "Pasar Lokal",
  "purchased_by": "<existing-user-uuid>",
  "note": "Tambahan 50 porsi",
  "prices": [
    {
      "material_id": "<uuid>",
      "unit_price": "32000.00"
    }
  ]
}
```

Rules:
- Scheduled menu row is locked to serialize concurrent portion increases.
- Current effective portions are the latest `additional_requirements.new_portions`, or original `scheduled_menus.planned_portions` when none exists.
- `new_portions` must be greater than current effective portions.
- Client does not submit additional quantities.
- Server calculates each material's additional quantity from scheduled-menu snapshot recipe × portion delta.
- Every calculated material must have exactly one price entry.
- Creates `additional_requirements`, `additional_requirement_items`, direct purchase, stock `IN`, and `ADDITIONAL_REQUIREMENT` movements atomically.
- Original procurement gross/net snapshot is not rewritten.

### Direct Purchase Read
- `GET /api/v1/direct-purchases/:id`
- `GET /api/v1/scheduled-menus/:id/direct-purchases`
- Status: implemented, latest CI validation in progress.

## Material Usage
Planned:
- actual material usage draft/submit
- Chef + Accountant approval/rejection
- revision/version invalidation
- stock OUT only after both approvals for current version

## Stock Opname
Planned:
- physical stock capture
- difference calculation
- adjustment request
- Chef + Accountant approval
- approved adjustment movement

## Important Business Errors
Current categories:
- invalid input -> `400`
- resource not found -> `404`
- invalid transition/conflict -> `409`
- `PO_CANCEL_DEADLINE_PASSED` -> `409`
- `INSUFFICIENT_STOCK` -> `422`
- `SHORTAGE_QTY_EXCEEDED` -> `422`
- PostgreSQL unique violation -> `409`

Still planned:
- `APPROVAL_VERSION_STALE`
- `INVALID_APPROVER_ROLE`
