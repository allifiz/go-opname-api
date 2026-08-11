# Decisions V1

This file records decisions that should not be reopened by an agent unless a new requirement explicitly supersedes them.

## D-001: Approved `rapat` flow is not routinely edited
The approved meeting document is treated as the business-flow reference. Routine implementation updates happen in repository docs, not by rewriting the approved flow document.

## D-002: Work directly on `main`
The project is being built from scratch and routine implementation work is committed directly to `main` while repository rules allow it.

## D-003: Documentation follows code changes
- `docs/progress.md` must be updated for every codebase change.
- Other docs must be updated whenever their domain is affected.

## D-004: Vendor is not master data
Supplier/vendor/source information is stored per PO item or transaction. Different materials may use different vendors and no vendor master is required in V1.

## D-005: Price is fixed from PO agreement
The agreed unit price does not change at receipt in V1.

## D-006: Over-delivery is accepted
If vendor quantity exceeds ordered quantity, accept the full quantity into stock and record actual value from actual received quantity × fixed agreed price.

## D-007: Shortage purchase is bounded
`SHORTAGE` direct-purchase quantity may not exceed remaining shortage for the related PO item.

## D-008: Increased portions use direct purchase
If production portions increase after PO, do not create another PO. Use `ADDITIONAL_REQUIREMENT`.

## D-009: Wrong materials are rejected
There is no material substitution flow in V1.

## D-010: Quality rejection is out of scope
V1 does not model quality-based acceptance/rejection.

## D-011: Scheduled menus are snapshots
Scheduling copies the current template structure/materials so historical schedules are independent from future template edits.

## D-012: Gross requirement is immutable historical intent
Never replace gross production requirement with net procurement quantity.

## D-013: Existing stock is reserved during planning
Future-use stock allocation is reserved; reservation is not a physical stock movement.

## D-014: H-1 re-check differs from procurement stock check
Procurement stock check determines net procurement. H-1 re-check determines whether an existing PO item can be cancelled.

## D-015: PO cancellation is item-level
H-1 cancellation is item-level, requires sufficient replacement stock, and stores a reason.

## D-016: No payment workflow
The application stores procurement/receipt financial evidence but does not execute payments.

## D-017: Dual approval for usage
Material usage requires Chef + Accountant approval before stock OUT.

## D-018: Dual approval for stock adjustment
Stock adjustment requires Chef + Accountant approval before stock movement.

## D-019: Revision invalidates approvals
Material edits increment entity version and require fresh approvals while preserving old approval history.

## D-020: Reject returns workflow for revision
Rejected approval entities enter `NEEDS_REVISION`.

## D-021: Negative stock is prohibited
Any stock OUT that would make inventory negative fails atomically.

## D-022: Stock movement is the audit ledger
Every physical stock change creates a movement; `material_stocks` is the current snapshot.

## D-023: No FIFO/batch in V1
No FIFO allocation or per-batch inventory tracking in V1.

## D-024: Roles and units use master tables
Do not use PostgreSQL enums for mutable role/unit master data.

## D-025: Pragmatic Go layering
Use `handler -> service -> repository -> sqlc/pgx`.

## D-026: One procurement request per scheduled menu
Rejection revises/resubmits the same procurement request lifecycle.

## D-027: One PO per procurement request
PO item quantities are derived server-side from verified `net_procurement_qty`.

## D-028: H-1 cancellation reserves replacement stock first
Replacement reservation and PO-item cancellation happen in one transaction.

## D-029: H-1 latest valid day
Cancellation is valid only while business date is strictly before `delivery_date`, using `Asia/Jakarta`.

## D-030: Receiving is cumulative per PO item
Receipt and over-delivery statuses use cumulative vendor-received quantity.

## D-031: Zero-quantity receipt is valid
A zero receipt records non-delivery and creates no stock movement.

## D-032: Positive receipt enters stock immediately
Accepted positive receipt quantity creates stock `IN` + `PO_RECEIPT` movement atomically.

## D-033: Receipt deal fields are server-derived
Material, unit, and agreed unit price are copied from the PO item.

## D-034: Receipt amount is database-derived
`actual_amount` is PostgreSQL-generated.

## D-035: Receipt documents store metadata, not blobs
Binary/object storage remains a separate infrastructure concern.

## D-036: SHORTAGE uses cumulative remaining shortage
Remaining shortage is `ordered_qty - cumulative vendor receipts - cumulative SHORTAGE direct purchases`, floored at zero. The PO item is locked before this calculation.

## D-037: Fully covered shortage marks item FULFILLED
When a shortage direct purchase exactly closes remaining shortage, the PO item becomes `FULFILLED` and PO header status is recalculated.

## D-038: Additional requirement quantity is server-calculated
For `ADDITIONAL_REQUIREMENT`, clients submit the new portion count and prices only. Material quantities are calculated from the scheduled-menu snapshot recipe × portion delta.

## D-039: Additional portion increases are cumulative
Current effective portions are the latest additional-requirement `new_portions`, falling back to the original scheduled-menu planned portions. The scheduled-menu row is locked while resolving/increasing this value.

## D-040: Additional requirement does not rewrite original procurement
The original procurement gross/existing/reserved/net snapshot remains historical. Additional production demand is represented separately in `additional_requirements` and direct-purchase records.

## D-041: Material usage is one versioned lifecycle per scheduled menu
A scheduled menu has one material-usage record. Revisions update that lifecycle, increment `version`, replace current usage items, and preserve historical approval rows.

## D-042: Usage planned quantity follows effective portions
Usage `planned_qty` is calculated from the scheduled-menu snapshot using the latest effective portion count, including later `ADDITIONAL_REQUIREMENT` increases. The client only supplies actual usage quantity.

## D-043: Usage approval identity is role-validated server-side
Approval identity must resolve to an active user whose role is `CHEF` or `AKUNTAN`. D-053 supersedes the old temporary request-body actor source; authenticated JWT identity is now authoritative.

## D-044: Approval uniqueness is role + entity version
A usage version may have at most one Chef decision and one Accountant decision. Approval rows are never deleted when a new version is created.

## D-045: Second usage approval applies inventory atomically
When both required roles have approved the current usage version, stock decrement, `OUT / MATERIAL_USAGE` movements, reservation consumption, second approval insertion, and final `APPROVED` status all belong to the same transaction. If any material is insufficient, the whole transaction rolls back.

## D-046: Usage completion consumes reservations
When approved usage is applied, active reservations for the scheduled menu/material are marked `CONSUMED`, including materials whose actual usage is zero. Reservation consumption itself is not a stock movement.

## D-047: Stock opname snapshots system quantity
Stock opname captures `system_qty` from the current locked `material_stocks` row and stores user-entered `physical_qty`. `difference_qty` is database-generated as physical minus system.

## D-048: Opname difference never auto-adjusts stock
A non-zero difference creates a DRAFT stock adjustment only. Inventory is unchanged until the adjustment receives both required approvals.

## D-049: Adjustment quantity is server-derived
Clients cannot submit `adjustment_qty`. It is copied from the opname item's generated difference. Revision changes `physical_qty` and reason, recalculates the difference, increments version, and leaves old approvals as history.

## D-050: Stock adjustment approval mirrors usage approval
Only active Chef and Accountant users may approve/reject. Decisions are unique by adjustment + role + entity version. Rejection moves the adjustment to `NEEDS_REVISION` without inventory effects.

## D-051: Second adjustment approval applies inventory atomically
The second current-version approval applies `ADJUSTMENT_IN` for positive difference or `ADJUSTMENT_OUT` for negative difference, writes `STOCK_ADJUSTMENT` movement, and marks the adjustment approved in one transaction. Negative stock causes the whole transaction, including the second approval, to roll back.

## D-052: Stock opname completes after all adjustments
A stock opname with differences becomes `COMPLETED` only after every generated adjustment is `APPROVED`. A matching opname has no adjustment records.

## D-053: JWT identity is authoritative for HTTP audit actors
All protected HTTP requests use a validated Bearer JWT. Operational actor fields that previously came from request bodies are ignored/overwritten by the authenticated user ID at the HTTP boundary. Clients cannot impersonate another actor by changing `verified_by`, `received_by`, `purchased_by`, `submitted_by`, `approver_id`, `performed_by`, or `cancelled_by` in JSON.

## D-054: JWT uses HS256 with explicit secret requirements
V1 uses HS256 implemented with Go standard-library HMAC/SHA-256. Tokens expire after 8 hours. `JWT_SECRET` is mandatory and must be at least 32 characters. Signature and expiry are validated before protected handlers execute.

## D-055: RBAC is enforced before business handlers
Role authorization is enforced at the HTTP route boundary. Ahli Gizi controls menu-planning writes; Pengawas controls warehouse/procurement operational writes; Akuntan controls procurement verification; Chef and Akuntan control dual-approval decisions. Authenticated reads remain broadly available until a narrower approved policy exists.

## D-056: No public default credential is seeded
Migrations do not create a reusable default user/password. Initial-user provisioning is a separate secure operational workflow to implement, avoiding a known credential embedded in repository history.
