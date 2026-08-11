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

Receiving supports cumulative `NOT_RECEIVED`, `PARTIAL_RECEIVED`, `RECEIVED`, and `OVER_RECEIVED`. Positive receipt quantity creates stock `IN` + `PO_RECEIPT` movement atomically. Zero quantity is valid and creates no movement.

## Direct Purchase
Implemented and green through CI run #44:
- `POST /api/v1/purchase-order-items/:id/direct-purchases/shortage`
- `POST /api/v1/scheduled-menus/:id/direct-purchases/additional-requirement`
- `GET /api/v1/direct-purchases/:id`
- `GET /api/v1/scheduled-menus/:id/direct-purchases`

`SHORTAGE` cannot exceed cumulative remaining shortage. `ADDITIONAL_REQUIREMENT` quantities are server-calculated from the scheduled-menu snapshot and effective portion delta. Both flows create stock `IN` and inventory movement atomically.

## Material Usage
Implemented, latest CI validation applies to this batch:
- `POST /api/v1/scheduled-menus/:id/material-usage`
- `GET /api/v1/material-usages/:id`
- `PUT /api/v1/material-usages/:id`
- `POST /api/v1/material-usages/:id/submit`
- `POST /api/v1/material-usages/:id/decision`

### Create / Revise
Create/update body:
```json
{
  "usage_date": "2026-08-12",
  "submitted_by": "<existing-user-uuid>",
  "items": [
    {
      "material_id": "<uuid>",
      "actual_qty": "12.5000"
    }
  ]
}
```

Rules:
- Usage is unique per scheduled menu.
- Input material set must exactly match the scheduled-menu snapshot material set.
- `planned_qty` is server-calculated from snapshot recipe × effective portions, including the latest `ADDITIONAL_REQUIREMENT` portion increase.
- Update is allowed only in `DRAFT` or `NEEDS_REVISION`.
- Every edit increments `version`, replaces current item rows, and leaves older approval rows as immutable history.

### Submit
`POST /api/v1/material-usages/:id/submit`
```json
{
  "submitted_by": "<existing-user-uuid>"
}
```

Transition: `DRAFT -> WAITING_APPROVAL`.

### Chef / Accountant Decision
`POST /api/v1/material-usages/:id/decision`
```json
{
  "approver_id": "<existing-user-uuid>",
  "decision": "APPROVED",
  "note": "optional"
}
```

Rules:
- Approver identity is resolved to its active user role.
- Only `CHEF` and `AKUNTAN` may decide.
- One decision per required role per entity version.
- `REJECTED` changes usage to `NEEDS_REVISION`.
- Two `APPROVED` decisions for the same current version trigger stock application atomically.
- Stock rows are locked before decrement.
- If any actual quantity would create negative stock, the entire approval/application transaction rolls back with `422`.
- Positive actual usage creates `OUT / MATERIAL_USAGE` stock movements.
- Active reservations for the scheduled menu/material are consumed when the approved usage is applied.
- Final status becomes `APPROVED` only after all stock effects succeed.

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
- PostgreSQL unique violation, including duplicate role decision for one version -> `409`
