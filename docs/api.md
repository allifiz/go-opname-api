# API Plan V1

This file tracks the HTTP contract surface. It is intentionally concise; detailed request/response schemas should live in code/OpenAPI once implemented.

## Conventions
- Base path: `/api/v1`.
- JSON request/response.
- Validation errors return `400` with `{ "error": "..." }`.
- Missing rows return `404`.
- Conflict/invalid state transitions return `409`.
- Insufficient stock for a business operation returns `422`.
- Unexpected database/application failures return `500`.
- Authentication/authorization is not implemented yet; audit user fields remain nullable until auth is added.
- `GET /health` remains outside `/api/v1`.

## Existing

### Health
- `GET /health`
- Status: implemented.

## Phase 1: Master Data

### Units
- `GET /api/v1/units`
- Status: implemented.

### Materials
- `GET /api/v1/materials`
- `GET /api/v1/materials/:id`
- `POST /api/v1/materials`
- `PUT /api/v1/materials/:id`
- Status: implemented.

## Phase 2: Period and Menu

### Periods
- `GET /api/v1/periods`
- `POST /api/v1/periods`
- Status: implemented.

Create body accepts `name` and `start_date`; the service derives `end_date = start_date + 13 days`.

### Menu Templates
- `GET /api/v1/menu-templates`
- `GET /api/v1/menu-templates/:id`
- `POST /api/v1/menu-templates`
- `PUT /api/v1/menu-templates/:id`
- Status: create/list/detail implemented; update still planned.

### Scheduled Menus
- `GET /api/v1/scheduled-menus/:id`
- `POST /api/v1/scheduled-menus`
- Status: implemented.

Creation validates `menu_date` inside the selected period and snapshots template components/materials transactionally.

## Phase 3: Procurement

### Procurement Stock Check
- `POST /api/v1/scheduled-menus/:id/procurement-stock-check`
- Status: implemented.

The operation aggregates gross requirement, locks material stock, accounts for active reservations, persists gross/existing/reserved/net quantities, and creates active reservations atomically.

### Procurement Request Read
- `GET /api/v1/procurement-requests/:id`
- `GET /api/v1/scheduled-menus/:id/procurement-requests`
- Status: implemented.

### Submit / Verify / Reject
- `POST /api/v1/procurement-requests/:id/submit`
- `POST /api/v1/procurement-requests/:id/verify`
- `POST /api/v1/procurement-requests/:id/reject`
- Status: implemented.

Until authentication exists, verify requires:
```json
{
  "verified_by": "<existing-user-uuid>"
}
```

### Generate Purchase Order
- `POST /api/v1/procurement-requests/:id/purchase-order`
- Status: implemented.
- Procurement request must be `VERIFIED`.
- One purchase order is allowed per procurement request.
- `ordered_qty` is always copied server-side from `net_procurement_qty`; clients cannot override it.
- Only positive net-procurement items are included.
- Every positive net-procurement item must be supplied exactly once in the request body.

Example body:
```json
{
  "delivery_date": "2026-08-20",
  "items": [
    {
      "procurement_request_item_id": "<uuid>",
      "supplier_name": "Supplier A",
      "agreed_unit_price": "27500.00"
    }
  ]
}
```

The generated PO starts in `VERIFIED` status and receives a server-generated `PO-YYYYMMDD-XXXXXXXXXX` number.

### Purchase Order Read
- `GET /api/v1/purchase-orders/:id`
- `GET /api/v1/scheduled-menus/:id/purchase-orders`
- Status: implemented.

### H-1 PO Item Cancellation
- `POST /api/v1/purchase-order-items/:id/cancel-h1`
- Status: implemented.
- Item must still be `WAITING`.
- Cancellation is allowed only before the delivery date, so the latest valid day is H-1.
- The service locks the PO item and material stock.
- It verifies enough currently-unreserved stock exists to replace the full ordered quantity.
- It creates an additional active stock reservation first, in the same transaction.
- Only after the reservation succeeds is the PO item changed to `CANCELLED`.
- Cancellation reason is fixed to `EXISTING_STOCK_SUFFICIENT` for this flow.

Until authentication exists, body requires:
```json
{
  "cancelled_by": "<existing-user-uuid>"
}
```

If unreserved stock is insufficient, the operation returns `422`. If the cancellation deadline has passed or item status is invalid, it returns `409`.

## Phase 4: Receiving and Direct Purchase
Planned capabilities:
- record PO receipt
- support `NOT_RECEIVED`, `PARTIAL_RECEIVED`, `RECEIVED`, `OVER_RECEIVED`
- record receipt documents/invoices
- record `SHORTAGE` direct purchase
- record `ADDITIONAL_REQUIREMENT` direct purchase
- expose documents/actual amount to Accountant

## Phase 5: Material Usage
Planned capabilities:
- draft/submit actual usage
- Chef approval/rejection
- Accountant approval/rejection
- revision/resubmission
- apply stock OUT only when both approvals for the current entity version are valid

## Phase 6: Stock Opname
Planned capabilities:
- create post-production stock opname
- record physical quantities
- calculate differences
- create adjustment request when needed
- Chef + Accountant approval
- apply approved adjustment to inventory ledger

## Important Business Errors
Current categories include:
- invalid input -> `400`
- resource not found -> `404`
- conflict / invalid transition -> `409`
- `PO_CANCEL_DEADLINE_PASSED` -> `409`
- `INSUFFICIENT_STOCK` -> `422`
- PostgreSQL unique violation -> `409`

Still planned:
- `SHORTAGE_QTY_EXCEEDED`
- `APPROVAL_VERSION_STALE`
- `INVALID_APPROVER_ROLE`
- `STOCK_ALREADY_RESERVED`
