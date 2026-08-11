# API Plan V1

This file tracks the HTTP contract surface. It is intentionally concise; detailed request/response schemas should live in code/OpenAPI once implemented.

## Conventions
- Base path: `/api/v1`.
- JSON request/response.
- Validation errors return `400` with `{ "error": "..." }`.
- Missing rows return `404`.
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

Create body:
```json
{
  "name": "Dada Ayam",
  "unit_id": "<uuid>"
}
```

Update body:
```json
{
  "name": "Dada Ayam",
  "unit_id": "<uuid>",
  "is_active": true
}
```

## Phase 2: Period and Menu

### Periods
- `GET /api/v1/periods`
- `POST /api/v1/periods`
- Status: implemented.

Create body only accepts `start_date`; the service derives `end_date = start_date + 13 days` to preserve the approved 14-day period invariant.

```json
{
  "name": "Periode Agustus 1",
  "start_date": "2026-08-17"
}
```

### Menu Templates
- `GET /api/v1/menu-templates`
- `GET /api/v1/menu-templates/:id`
- `POST /api/v1/menu-templates`
- `PUT /api/v1/menu-templates/:id`
- Status: create/list/detail implemented; update is still planned.

Create is transactional and accepts nested components/materials:
```json
{
  "name": "Ayam Katsu",
  "description": "Template menu",
  "components": [
    {
      "name": "Lauk",
      "sort_order": 1,
      "materials": [
        {
          "material_id": "<uuid>",
          "qty_per_portion": "0.0800",
          "unit_id": "<uuid>"
        }
      ]
    }
  ]
}
```

`qty_per_portion` is represented as a JSON string to avoid floating-point precision loss before PostgreSQL `NUMERIC(18,4)`.

### Scheduled Menus
- `GET /api/v1/scheduled-menus/:id`
- `POST /api/v1/scheduled-menus`
- Status: implemented.

Creation validates that `menu_date` is inside the selected period, then transactionally snapshots the selected template components and materials.

```json
{
  "period_id": "<uuid>",
  "menu_template_id": "<uuid>",
  "menu_date": "2026-08-17",
  "planned_portions": 500
}
```

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

Current core handlers use explicit validation messages. Stable procurement/stock error identifiers will be added with the procurement service.
