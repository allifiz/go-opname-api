# Project Progress

## Current Phase
Material usage, versioned Chef + Accountant approval, and atomic stock OUT are implemented and green in GitHub Actions. Next focus: stock opname + dual-approved adjustment.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, and material usage implemented.
- `000008_create_material_usage.sql` implemented.
- Usage lifecycle: `DRAFT`, `WAITING_APPROVAL`, `APPROVED`, `NEEDS_REVISION`.
- Usage `planned_qty` is server-derived from scheduled snapshot × latest effective portions, including `ADDITIONAL_REQUIREMENT` increases.
- Usage revision increments `version`, replaces current items, and preserves old approvals as history.
- Chef + Accountant approvals are unique by usage + role + entity version.
- Only active `CHEF` / `AKUNTAN` users may decide.
- Rejection returns usage to `NEEDS_REVISION` without stock changes.
- The second current-version approval atomically applies stock OUT, creates `MATERIAL_USAGE` movements, consumes active reservations, and marks usage `APPROVED`.
- Insufficient stock rolls back the entire second-approval transaction.
- Material usage endpoints are implemented and documented.
- GitHub Actions run #55 passed migrations, rollback/re-apply, `sqlc generate`, `go test ./...`, `go build ./...`, and generated-code synchronization.

## Implemented Migrations
- `000001_create_foundation.sql`
- `000002_create_menu.sql`
- `000003_create_inventory.sql`
- `000004_create_procurement.sql`
- `000005_purchase_order_constraints.sql`
- `000006_create_receiving.sql`
- `000007_create_direct_purchase.sql`
- `000008_create_material_usage.sql`

## Latest HTTP Endpoints
- `POST /api/v1/scheduled-menus/:id/material-usage`
- `GET /api/v1/material-usages/:id`
- `PUT /api/v1/material-usages/:id`
- `POST /api/v1/material-usages/:id/submit`
- `POST /api/v1/material-usages/:id/decision`

## Validation Status
GREEN through GitHub Actions run #55.

## In Progress
- Prepare stock opname schema and adjustment workflow.

## Next
1. Implement `000009_stock_opname.sql`.
2. Capture physical stock and server-calculated differences.
3. Do not auto-adjust stock when a difference exists.
4. Add versioned Chef + Accountant approval for adjustments.
5. Apply approved `ADJUSTMENT_IN` / `ADJUSTMENT_OUT` atomically with negative-stock protection.
6. Add focused concurrency/transaction tests.
7. Add menu-template update without mutating historical scheduled-menu snapshots.
8. Add authentication/RBAC after the core inventory cycle is complete.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authentication/authorization is not implemented yet.
- Receipt document object storage provider is not implemented yet.
- No application payment workflow is required.

## Notes for Next Agent
Read before changing behavior/schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md`

Do not redesign approved business rules unless a new requirement explicitly supersedes them.
