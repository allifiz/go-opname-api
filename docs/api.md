# API Plan V1

This file tracks the HTTP contract surface. It is intentionally concise; detailed request/response schemas should live in code/OpenAPI once implemented.

## Conventions
- Base path: `/api/v1`.
- JSON request/response.
- Business validation failures should return explicit error codes/messages.
- Role authorization must follow approved actor responsibilities.
- `GET /health` remains outside `/api/v1`.

## Existing

### Health
- `GET /health`
- Purpose: application health check.
- Status: implemented.

## Phase 1: Master Data

### Units
- `GET /api/v1/units`
- Status: planned.

### Materials
- `GET /api/v1/materials`
- `POST /api/v1/materials`
- `PUT /api/v1/materials/:id`
- Status: planned.

## Phase 2: Period and Menu

### Periods
- `GET /api/v1/periods`
- `POST /api/v1/periods`
- Status: planned.

### Menu Templates
- `GET /api/v1/menu-templates`
- `GET /api/v1/menu-templates/:id`
- `POST /api/v1/menu-templates`
- `PUT /api/v1/menu-templates/:id`
- Status: planned.

### Scheduled Menus
- `GET /api/v1/scheduled-menus/:id`
- `POST /api/v1/scheduled-menus`
- Status: planned.

Creation must snapshot the selected menu template components/materials.

## Phase 3: Procurement
Planned capabilities:
- calculate/review gross requirements
- procurement stock check
- create/update stock reservation
- submit net procurement for Accountant verification
- Accountant verify/reject
- generate PO from verified net procurement
- H-1 stock re-check
- cancel individual PO item with reason

Exact endpoint paths will be finalized when implementing this module.

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
The service layer should expose stable business error identifiers for at least:
- `INSUFFICIENT_STOCK`
- `SHORTAGE_QTY_EXCEEDED`
- `PO_CANCEL_DEADLINE_PASSED`
- `APPROVAL_VERSION_STALE`
- `INVALID_APPROVER_ROLE`
- `INVALID_STATUS_TRANSITION`
- `STOCK_ALREADY_RESERVED`

HTTP status mapping will be finalized during handler implementation; business-rule failures should not become generic `500` responses.
