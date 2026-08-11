package service

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    db "github.com/allifiz/go-opname-api/internal/database/generated"
    "github.com/allifiz/go-opname-api/internal/repository"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgtype"
)

type MaterialUsageService struct { store *repository.Store }

func NewMaterialUsageService(store *repository.Store) *MaterialUsageService { return &MaterialUsageService{store: store} }

type MaterialUsageItemInput struct {
    MaterialID string `json:"material_id"`
    ActualQty  string `json:"actual_qty"`
}

type CreateMaterialUsageInput struct {
    UsageDate   string                   `json:"usage_date"`
    SubmittedBy string                   `json:"submitted_by"`
    Items       []MaterialUsageItemInput `json:"items"`
}

type DecideMaterialUsageInput struct {
    ApproverID string `json:"approver_id"`
    Decision   string `json:"decision"`
    Note       string `json:"note"`
}

func parseUsageActuals(items []MaterialUsageItemInput) (map[string]pgtype.Numeric, error) {
    if len(items) == 0 { return nil, fmt.Errorf("%w: usage items are required", ErrInvalidInput) }
    actualByMaterial := make(map[string]pgtype.Numeric, len(items))
    for _, in := range items {
        id, err := parseUUID(in.MaterialID); if err != nil { return nil, err }
        if _, ok := actualByMaterial[id.String()]; ok { return nil, fmt.Errorf("%w: duplicate material_id", ErrInvalidInput) }
        qty, err := parseNumeric(in.ActualQty); if err != nil || !numericNonNegative(qty) { return nil, fmt.Errorf("%w: actual_qty must be non-negative numeric", ErrInvalidInput) }
        actualByMaterial[id.String()] = qty
    }
    return actualByMaterial, nil
}

func writeUsageItems(ctx context.Context, q *db.Queries, usageID, scheduledMenuID pgtype.UUID, actualByMaterial map[string]pgtype.Numeric) error {
    requirements, err := q.GetScheduledMenuUsageRequirements(ctx, scheduledMenuID); if err != nil { return err }
    if len(requirements) == 0 || len(requirements) != len(actualByMaterial) { return fmt.Errorf("%w: usage items must exactly match scheduled menu materials", ErrInvalidInput) }
    for _, req := range requirements {
        actual, ok := actualByMaterial[req.MaterialID.String()]; if !ok { return fmt.Errorf("%w: usage items must exactly match scheduled menu materials", ErrInvalidInput) }
        if _, err := q.CreateMaterialUsageItem(ctx, db.CreateMaterialUsageItemParams{MaterialUsageID: usageID, MaterialID: req.MaterialID, PlannedQty: req.PlannedQty, ActualQty: actual, UnitID: req.UnitID}); err != nil { return err }
    }
    return nil
}

func (s *MaterialUsageService) Create(ctx context.Context, scheduledMenuIDValue string, input CreateMaterialUsageInput) (map[string]any, error) {
    scheduledMenuID, err := parseUUID(scheduledMenuIDValue); if err != nil { return nil, err }
    submittedBy, err := parseUUID(input.SubmittedBy); if err != nil { return nil, fmt.Errorf("%w: submitted_by is required until authentication is implemented", ErrInvalidInput) }
    usageDate, err := parseDate(input.UsageDate); if err != nil { return nil, err }
    actualByMaterial, err := parseUsageActuals(input.Items); if err != nil { return nil, err }

    var usage db.MaterialUsage
    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        if _, err := q.GetMaterialUsageByScheduledMenu(ctx, scheduledMenuID); err == nil { return fmt.Errorf("%w: material usage already exists for scheduled menu", ErrConflict) } else if !errors.Is(err, pgx.ErrNoRows) { return err }
        usage, err = q.CreateMaterialUsage(ctx, db.CreateMaterialUsageParams{ScheduledMenuID: scheduledMenuID, UsageDate: usageDate, SubmittedBy: submittedBy}); if err != nil { return err }
        return writeUsageItems(ctx, q, usage.ID, scheduledMenuID, actualByMaterial)
    })
    if err != nil { return nil, err }
    return s.getByUUID(ctx, usage.ID)
}

func (s *MaterialUsageService) Update(ctx context.Context, idValue string, input CreateMaterialUsageInput) (map[string]any, error) {
    id, err := parseUUID(idValue); if err != nil { return nil, err }
    submittedBy, err := parseUUID(input.SubmittedBy); if err != nil { return nil, fmt.Errorf("%w: submitted_by is required until authentication is implemented", ErrInvalidInput) }
    usageDate, err := parseDate(input.UsageDate); if err != nil { return nil, err }
    actualByMaterial, err := parseUsageActuals(input.Items); if err != nil { return nil, err }

    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        current, err := q.LockMaterialUsage(ctx, id); if err != nil { return err }
        if current.Status != db.MaterialUsageStatusDRAFT && current.Status != db.MaterialUsageStatusNEEDSREVISION { return fmt.Errorf("%w: only DRAFT or NEEDS_REVISION usage can be edited", ErrInvalidTransition) }
        if _, err := q.UpdateMaterialUsageForRevision(ctx, db.UpdateMaterialUsageForRevisionParams{ID: id, UsageDate: usageDate, SubmittedBy: submittedBy}); err != nil { return err }
        if err := q.DeleteMaterialUsageItems(ctx, id); err != nil { return err }
        return writeUsageItems(ctx, q, id, current.ScheduledMenuID, actualByMaterial)
    })
    if err != nil { return nil, err }
    return s.getByUUID(ctx, id)
}

func (s *MaterialUsageService) Submit(ctx context.Context, idValue, submittedByValue string) (db.MaterialUsage, error) {
    id, err := parseUUID(idValue); if err != nil { return db.MaterialUsage{}, err }
    by, err := parseUUID(submittedByValue); if err != nil { return db.MaterialUsage{}, fmt.Errorf("%w: submitted_by is required", ErrInvalidInput) }
    usage, err := s.store.SubmitMaterialUsage(ctx, db.SubmitMaterialUsageParams{ID: id, SubmittedBy: by})
    if errors.Is(err, pgx.ErrNoRows) { return db.MaterialUsage{}, fmt.Errorf("%w: usage must be DRAFT", ErrInvalidTransition) }
    return usage, err
}

func (s *MaterialUsageService) Decide(ctx context.Context, idValue string, input DecideMaterialUsageInput) (map[string]any, error) {
    id, err := parseUUID(idValue); if err != nil { return nil, err }
    approverID, err := parseUUID(input.ApproverID); if err != nil { return nil, fmt.Errorf("%w: approver_id is required", ErrInvalidInput) }
    decisionText := strings.ToUpper(strings.TrimSpace(input.Decision))
    var decision db.ApprovalDecision
    switch decisionText { case "APPROVED": decision = db.ApprovalDecisionAPPROVED; case "REJECTED": decision = db.ApprovalDecisionREJECTED; default: return nil, fmt.Errorf("%w: decision must be APPROVED or REJECTED", ErrInvalidInput) }

    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        usage, err := q.LockMaterialUsage(ctx, id); if err != nil { return err }
        if usage.Status != db.MaterialUsageStatusWAITINGAPPROVAL { return fmt.Errorf("%w: usage must be WAITING_APPROVAL", ErrInvalidTransition) }
        user, err := q.GetUserWithRole(ctx, approverID); if err != nil { return err }
        if !user.IsActive { return fmt.Errorf("%w: approver user is inactive", ErrInvalidInput) }
        var role db.UsageApproverRole
        switch user.RoleCode { case "CHEF": role = db.UsageApproverRoleCHEF; case "AKUNTAN": role = db.UsageApproverRoleAKUNTAN; default: return fmt.Errorf("%w: approver must be CHEF or AKUNTAN", ErrInvalidInput) }
        note := pgtype.Text{}
        if strings.TrimSpace(input.Note) != "" { note = pgtype.Text{String: strings.TrimSpace(input.Note), Valid: true} }
        if _, err := q.CreateMaterialUsageApproval(ctx, db.CreateMaterialUsageApprovalParams{MaterialUsageID: usage.ID, ApproverRole: role, ApproverID: approverID, EntityVersion: usage.Version, Status: decision, Note: note}); err != nil { return err }
        if decision == db.ApprovalDecisionREJECTED { _, err = q.MarkMaterialUsageNeedsRevision(ctx, usage.ID); return err }
        approvedCount, err := q.CountApprovedMaterialUsageRolesForVersion(ctx, db.CountApprovedMaterialUsageRolesForVersionParams{MaterialUsageID: usage.ID, EntityVersion: usage.Version}); if err != nil { return err }
        if approvedCount < 2 { return nil }
        items, err := q.ListMaterialUsageItems(ctx, usage.ID); if err != nil { return err }
        for _, item := range items {
            if numericPositive(item.ActualQty) {
                if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{MaterialID: item.MaterialID, UnitID: item.UnitID}); err != nil { return err }
                stock, err := q.LockMaterialStock(ctx, item.MaterialID); if err != nil { return err }
                if stock.UnitID != item.UnitID { return fmt.Errorf("%w: stock unit differs from usage unit", ErrConflict) }
                if _, err := q.DecreaseMaterialStockIfSufficient(ctx, db.DecreaseMaterialStockIfSufficientParams{SubtractQty: item.ActualQty, MaterialID: item.MaterialID, UnitID: item.UnitID}); errors.Is(err, pgx.ErrNoRows) { return fmt.Errorf("%w: insufficient stock for material usage", ErrInsufficientStock) } else if err != nil { return err }
                if _, err := q.CreateStockMovement(ctx, db.CreateStockMovementParams{MaterialID: item.MaterialID, MovementType: db.StockMovementTypeOUT, ReferenceType: db.StockReferenceTypeMATERIALUSAGE, ReferenceID: item.ID, Qty: item.ActualQty, UnitID: item.UnitID, MovementDate: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}, CreatedBy: approverID}); err != nil { return err }
            }
            if err := q.ConsumeActiveReservationsByScheduledMenuMaterial(ctx, db.ConsumeActiveReservationsByScheduledMenuMaterialParams{ScheduledMenuID: usage.ScheduledMenuID, MaterialID: item.MaterialID}); err != nil { return err }
        }
        _, err = q.MarkMaterialUsageApproved(ctx, usage.ID)
        return err
    })
    if err != nil { return nil, err }
    return s.getByUUID(ctx, id)
}

func (s *MaterialUsageService) getByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
    usage, err := s.store.GetMaterialUsage(ctx, id); if err != nil { return nil, err }
    items, err := s.store.ListMaterialUsageItems(ctx, id); if err != nil { return nil, err }
    approvals, err := s.store.ListMaterialUsageApprovals(ctx, id); if err != nil { return nil, err }
    return map[string]any{"material_usage": usage, "items": items, "approvals": approvals}, nil
}

func (s *MaterialUsageService) Get(ctx context.Context, idValue string) (map[string]any, error) { id, err := parseUUID(idValue); if err != nil { return nil, err }; return s.getByUUID(ctx, id) }
