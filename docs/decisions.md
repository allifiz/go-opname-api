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
If vendor quantity exceeds ordered quantity:
- accept the full actual quantity;
- put the full quantity into stock;
- status is `OVER_RECEIVED`;
- pay/record actual value as `received_qty * agreed_unit_price`.

## D-007: Shortage purchase is bounded
`SHORTAGE` direct-purchase quantity may not exceed the remaining shortage for the related PO item.

## D-008: Increased portions use direct purchase
If portions/production need increase after PO, do not create another PO. Create a direct purchase with type `ADDITIONAL_REQUIREMENT`.

## D-009: Wrong materials are rejected
There is no material substitution flow in V1.

## D-010: Quality rejection is out of scope
V1 does not model quality-based acceptance/rejection. Receipt is concerned with actual material and quantity received.

## D-011: Scheduled menus are snapshots
A menu template is reusable. Scheduling copies the current template structure/materials so historical schedules are independent from future template edits.

## D-012: Gross requirement is immutable historical intent
Never replace gross production requirement with net procurement quantity. Gross requirement, stock offset/allocation, and net procurement are separately auditable.

## D-013: Existing stock is reserved during planning
Stock used to offset a future procurement requirement must be reserved so it cannot be mathematically reused by another requirement.

Reservation is not a stock movement.

## D-014: H-1 re-check is different from procurement stock check
- Procurement stock check happens before Accountant verification and determines net procurement.
- H-1 stock re-check happens after PO formation and determines whether a PO item can be cancelled.

## D-015: PO cancellation is item-level
Pengawas Bahan Baku may cancel an individual PO item at maximum H-1 when stock is sufficient. Reason must be stored. Historical gross/net calculation remains unchanged.

## D-016: No payment workflow
The application does not execute or approve payments. Pengawas uploads invoices/supporting evidence and Accountant can view them.

## D-017: Dual approval for usage
Material usage requires both Chef and Accountant approval before stock OUT occurs.

## D-018: Dual approval for stock adjustment
Stock adjustment requires both Chef and Accountant approval before adjustment movement occurs.

## D-019: Revision invalidates approvals
When an approved/submitted entity is materially edited:
- increment entity version;
- previous approvals remain historical;
- previous approvals are no longer valid;
- fresh approvals are required.

## D-020: Reject returns workflow for revision
If one required approver rejects, the entity enters `NEEDS_REVISION` rather than partially applying its effect.

## D-021: Negative stock is prohibited
Any stock OUT that would make inventory negative must fail atomically.

## D-022: Stock movement is the audit ledger
Every real stock change creates a movement. `material_stocks` is only the current read-optimized snapshot.

## D-023: No FIFO/batch in V1
No FIFO allocation or per-batch inventory tracking is implemented in V1.

## D-024: Roles and units use master tables
Do not use PostgreSQL enums for mutable role/unit master data. Use tables seeded with approved V1 values.

## D-025: Pragmatic Go layering
Use `handler -> service -> repository -> sqlc/pgx`. Do not add architectural layers without an actual need.

## D-026: One procurement request per scheduled menu
A scheduled menu has one procurement request lifecycle. Accountant rejection revises/resubmits that same request instead of creating a second request. This prevents duplicate stock reservations and duplicate net-procurement snapshots for the same scheduled menu.

## D-027: One PO per procurement request
A verified procurement request creates at most one PO document in V1. PO item quantities are derived server-side from verified `net_procurement_qty`; clients only provide delivery date, supplier name, and fixed agreed unit price.

## D-028: H-1 cancellation reserves replacement stock first
An H-1 cancellation caused by sufficient existing stock must create an additional active reservation for the full cancelled PO quantity in the same transaction before changing the item to `CANCELLED`. If the reservation cannot be made, cancellation fails.

## D-029: H-1 latest valid day
For the H-1 cancellation flow, cancellation is valid only while the business date is strictly before `delivery_date`. Delivery day and later are rejected. Business-date evaluation currently uses `Asia/Jakarta`.
