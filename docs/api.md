# API Plan V1

This file tracks the HTTP contract surface. Detailed schemas belong in code/OpenAPI once implemented.

## Conventions
- Base path: `/api/v1`.
- JSON request/response.
- `GET /health` and `POST /api/v1/auth/login` are public.
- Every other `/api/v1` endpoint requires `Authorization: Bearer <JWT>`.
- Missing/invalid/expired token -> `401`.
- Authenticated role not permitted -> `403`.
- Validation errors -> `400`.
- Missing rows -> `404`.
- Conflict/invalid transition -> `409`.
- Business quantity/stock violations -> `422`.
- Unexpected failures -> `500`.
- Operational audit identity is derived from JWT and is not trusted from request-body actor fields.
- Quantities and prices are represented as JSON strings when mapped to PostgreSQL `NUMERIC`.

## Authentication

### Login
`POST /api/v1/auth/login`

```json
{
  "email": "akuntan@example.com",
  "password": "secret"
}
```

Success returns an HS256 Bearer token valid for 8 hours plus the authenticated user summary. Passwords are verified with bcrypt against `users.password_hash`.

Token claims:
- `sub`: user UUID
- `role`: current role code at login
- `email`
- `exp`

`JWT_SECRET` is required at runtime and must be at least 32 characters.

### Current Actor
`GET /api/v1/auth/me`

Returns the user ID, role, and email resolved from the validated token.

### Initial User Provisioning
No public default credential is seeded by migrations. Initial users must be provisioned securely outside the public login endpoint until a dedicated provisioning/admin workflow is implemented.

## RBAC

Write permissions currently enforced at the HTTP boundary:
- `AHLI_GIZI`: create periods, menu templates, and scheduled menus.
- `AHLI_GIZI` or `PENGAWAS_BAHAN_BAKU`: create/update materials.
- `PENGAWAS_BAHAN_BAKU`: procurement stock check/submission, PO generation/H-1 cancellation, receiving, direct purchases, material usage entry/revision/submission, stock opname, adjustment revision/submission.
- `AKUNTAN`: procurement verification/rejection.
- `CHEF` or `AKUNTAN`: material-usage and stock-adjustment decisions.

Authenticated read endpoints are currently available to all authenticated roles unless a narrower rule is added later. `KEPALA_SPPG` operational permissions remain TBD.

## Master / Menu
Implemented:
- `GET /health` (public)
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

`verified_by` and `cancelled_by` are derived from JWT. PO ordered quantity remains server-derived from verified `net_procurement_qty`.

## Receiving
Implemented:
- `POST /api/v1/purchase-orders/:id/receipts`
- `GET /api/v1/purchase-orders/:id/receipts`
- `GET /api/v1/receipts/:id`

For create receipt, `received_by` is derived from JWT. Receiving remains cumulative and positive receipt quantity creates stock `IN / PO_RECEIPT` atomically.

## Direct Purchase
Implemented:
- `POST /api/v1/purchase-order-items/:id/direct-purchases/shortage`
- `POST /api/v1/scheduled-menus/:id/direct-purchases/additional-requirement`
- `GET /api/v1/direct-purchases/:id`
- `GET /api/v1/scheduled-menus/:id/direct-purchases`

`purchased_by` is derived from JWT. `SHORTAGE` cannot exceed cumulative remaining shortage. `ADDITIONAL_REQUIREMENT` quantities are server-calculated from scheduled-menu snapshot recipe and effective portion delta.

## Material Usage
Implemented:
- `POST /api/v1/scheduled-menus/:id/material-usage`
- `GET /api/v1/material-usages/:id`
- `PUT /api/v1/material-usages/:id`
- `POST /api/v1/material-usages/:id/submit`
- `POST /api/v1/material-usages/:id/decision`

Create/update body no longer needs an audit actor:
```json
{
  "usage_date": "2026-08-12",
  "items": [
    {
      "material_id": "<uuid>",
      "actual_qty": "12.5000"
    }
  ]
}
```

Rules:
- Pengawas creates/revises/submits usage.
- JWT user becomes `submitted_by`.
- Chef or Accountant makes a decision; JWT user becomes `approver_id`.
- Usage requires both roles to approve the same entity version before stock OUT.
- Negative stock is rejected atomically.

Decision body:
```json
{
  "decision": "APPROVED",
  "note": "optional"
}
```

## Stock Opname
Implemented:
- `POST /api/v1/scheduled-menus/:id/stock-opname`
- `GET /api/v1/stock-opnames/:id`
- `GET /api/v1/stock-adjustments/:id`
- `PUT /api/v1/stock-adjustments/:id`
- `POST /api/v1/stock-adjustments/:id/submit`
- `POST /api/v1/stock-adjustments/:id/decision`

Create example:
```json
{
  "opname_date": "2026-08-12",
  "items": [
    {
      "material_id": "<uuid>",
      "physical_qty": "8.5000",
      "reason": "Selisih hasil hitung fisik"
    }
  ]
}
```

`performed_by` is derived from the Pengawas JWT.

Revise adjustment:
```json
{
  "physical_qty": "9.0000",
  "reason": "Hitung ulang fisik"
}
```

`submitted_by` is derived from JWT. Client never sends `adjustment_qty`; it remains server-derived from the opname difference.

Decision:
```json
{
  "decision": "APPROVED",
  "note": "optional"
}
```

JWT identity becomes `approver_id`. Only Chef and Accountant may decide, and two approvals for the same current version are required before the adjustment changes inventory.

## Important Business Errors
Current categories:
- invalid/missing credentials -> `401`
- authenticated role denied -> `403`
- invalid input -> `400`
- resource not found -> `404`
- invalid transition/conflict -> `409`
- `PO_CANCEL_DEADLINE_PASSED` -> `409`
- `INSUFFICIENT_STOCK` -> `422`
- `SHORTAGE_QTY_EXCEEDED` -> `422`
- PostgreSQL unique violation, including duplicate role decision for one version -> `409`
