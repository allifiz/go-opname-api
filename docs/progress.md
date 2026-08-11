# Project Progress

## Current Phase
Foundation, Menu, and Inventory database foundation.

## Last Updated
2026-08-11

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document is treated as the business-flow reference and is not modified from this repository workflow.

## Completed
- Business requirement V1 discussed and approved.
- Menu template and scheduled-menu snapshot model agreed.
- Procurement stock check and H-1 re-check separated conceptually.
- Gross requirement, usable existing stock, and net procurement separated for audit.
- Over-received vendor goods are accepted and paid based on actual received quantity.
- Direct purchase types agreed: `SHORTAGE` and `ADDITIONAL_REQUIREMENT`.
- Material usage and stock adjustment require Chef + Akuntan approval.
- Negative stock is rejected.
- Stock movement ledger is the inventory audit source of truth.
- Stock reservation concept agreed to prevent existing stock from being allocated twice.
- Repository documentation structure established on `main`:
  - `docs/progress.md`
  - `docs/requirements.md`
  - `docs/architecture.md`
  - `docs/database.md`
  - `docs/api.md`
  - `docs/decisions.md`

## In Progress
- Replace starter foundation schema with approved V1 master data.
- Add menu and scheduled-menu schema.
- Add inventory stock, movement, and reservation schema.

## Next
1. Finalize foundation migration.
2. Add menu migration.
3. Add inventory migration.
4. Add sqlc queries for master/material and menu reads/writes.
5. Run `sqlc generate` locally.
6. Implement material and menu services/endpoints.
7. Continue with procurement schema and workflow.

## Blockers / TBD
- `KEPALA_SPPG` exists as a role, but operational permissions are still TBD.
- No application payment workflow is required.

## Latest Decisions
- Vendor is not master data; supplier/source name is stored per transaction/item.
- Fixed unit price is agreed before PO and does not change at receipt.
- Vendor over-delivery is accepted in full.
- Shortage direct purchase cannot exceed remaining shortage.
- Additional portions after PO use direct purchase type `ADDITIONAL_REQUIREMENT`, not a new PO.
- Wrong material delivery is rejected; no substitution flow.
- Material quality rejection is outside V1 scope.
- No FIFO or stock-batch tracking in V1.
- No negative stock.
- Approval becomes invalid when approved data is revised; resubmission requires fresh approval.
- Routine development is performed directly on `main` while repository rules permit direct writes.

## Changed Files
- `docs/progress.md`
- `docs/requirements.md`
- `docs/architecture.md`
- `docs/database.md`
- `docs/api.md`
- `docs/decisions.md`

## Notes for Next Agent
Read these before changing behavior or schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md` when touching HTTP contracts.

Do not redesign an approved business rule merely because another design looks cleaner. Change agreed behavior only when a new requirement explicitly supersedes it.
