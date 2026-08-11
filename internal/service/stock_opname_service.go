package service

import (
    "context"
    "errors"
    "fmt"
    "math/big"
    "strings"
    "time"

    db "github.com/allifiz/go-opname-api/internal/database/generated"
    "github.com/allifiz/go-opname-api/internal/repository"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgtype"
)

type StockOpnameService struct { store *repository.Store }

func NewStockOpnameService(store *repository.Store) *StockOpnameService { return &StockOpnameService{store: store} }

type StockOpnameItemInput struct {
    MaterialID  string `json:"material_id"`
    PhysicalQty string `json:"physical_qty"`
    Reason      string `json:"reason"`
}

type CreateStockOpnameInput struct {
    OpnameDate  string                 `json:"opname_date"`
    PerformedBy string                 `json:"performed_by"`
    Items       []StockOpnameItemInput `json:"items"`
}

type ReviseStockAdjustmentInput struct {
    PhysicalQty string `json:"physical_qty"`
    Reason      string `json:"reason"`
    SubmittedBy string `json:"submitted_by"`
}

type DecideStockAdjustmentInput struct {
    ApproverID string `json:"approver_id"`
    Decision   string `json:"decision"`
    Note       string `json:"note"`
}

func numericAbs(n pgtype.Numeric) pgtype.Numeric {
    out := n
    if n.Int != nil { out.Int = new(big.Int).Abs(new(big.Int).Set(n.Int)) }
    return out
}

func numericSign(n pgtype.Numeric) (int, error) {
    r, err := numericRat(n)
    if err != nil { return 0, err }
    return r.Sign(), nil
}

func (s *StockOpnameService) Create(ctx context.Context, scheduledMenuIDValue string, input CreateStockOpnameInput) (map[string]any, error) {
    scheduledMenuID, err := parseUUID(scheduledMenuIDValue); if err != nil { return nil, err }
    performedBy, err := parseUUID(input.PerformedBy); if err != nil { return nil, fmt.Errorf("%w: performed_by is required until authentication is implemented", ErrInvalidInput) }
    opnameDate, err := parseDate(input.OpnameDate); if err != nil { return nil, err }
    if len(input.Items) == 0 { return nil, fmt.Errorf("%w: opname items are required", ErrInvalidInput) }

    physical := make(map[string]StockOpnameItemInput, len(input.Items))
    parsed := make(map[string]pgtype.Numeric, len(input.Items))
    for _, item := range input.Items {
        id, err := parseUUID(item.MaterialID); if err != nil { return nil, err }
        key := id.String()
        if _, ok := physical[key]; ok { return nil, fmt.Errorf("%w: duplicate material_id", ErrInvalidInput) }
        qty, err := parseNumeric(item.PhysicalQty); if err != nil || !numericNonNegative(qty) { return nil, fmt.Errorf("%w: physical_qty must be non-negative numeric", ErrInvalidInput) }
        physical[key] = item
        parsed[key] = qty
    }

    var opname db.StockOpname
    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        if _, err := q.GetStockOpnameByScheduledMenu(ctx, scheduledMenuID); err == nil { return fmt.Errorf("%w: stock opname already exists for scheduled menu", ErrConflict) } else if !errors.Is(err, pgx.ErrNoRows) { return err }
        requirements, err := q.GetScheduledMenuUsageRequirements(ctx, scheduledMenuID); if err != nil { return err }
        if len(requirements) == 0 || len(requirements) != len(physical) { return fmt.Errorf("%w: opname items must exactly match scheduled menu materials", ErrInvalidInput) }
        opname, err = q.CreateStockOpname(ctx, db.CreateStockOpnameParams{ScheduledMenuID: scheduledMenuID, OpnameDate: opnameDate, PerformedBy: performedBy}); if err != nil { return err }
        hasDifference := false
        for _, req := range requirements {
            inputItem, ok := physical[req.MaterialID.String()]; if !ok { return fmt.Errorf("%w: opname items must exactly match scheduled menu materials", ErrInvalidInput) }
            if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{MaterialID: req.MaterialID, UnitID: req.UnitID}); err != nil { return err }
            stock, err := q.LockMaterialStock(ctx, req.MaterialID); if err != nil { return err }
            if stock.UnitID != req.UnitID { return fmt.Errorf("%w: stock unit differs from opname unit", ErrConflict) }
            opnameItem, err := q.CreateStockOpnameItem(ctx, db.CreateStockOpnameItemParams{StockOpnameID: opname.ID, MaterialID: req.MaterialID, SystemQty: stock.Qty, PhysicalQty: parsed[req.MaterialID.String()], UnitID: req.UnitID}); if err != nil { return err }
            sign, err := numericSign(opnameItem.DifferenceQty); if err != nil { return err }
            if sign != 0 {
                hasDifference = true
                reason := strings.TrimSpace(inputItem.Reason)
                if reason == "" { return fmt.Errorf("%w: reason is required for materials with stock difference", ErrInvalidInput) }
                if _, err := q.CreateStockAdjustment(ctx, db.CreateStockAdjustmentParams{StockOpnameItemID: opnameItem.ID, MaterialID: opnameItem.MaterialID, AdjustmentQty: opnameItem.DifferenceQty, Reason: reason, SubmittedBy: performedBy}); err != nil { return err }
            }
        }
        status := db.StockOpnameStatusMATCHED
        if hasDifference { status = db.StockOpnameStatusDIFFERENCEFOUND }
        _, err = q.UpdateStockOpnameStatus(ctx, db.UpdateStockOpnameStatusParams{ID: opname.ID, Status: status})
        return err
    })
    if err != nil { return nil, err }
    return s.getOpnameByUUID(ctx, opname.ID)
}

func (s *StockOpnameService) ReviseAdjustment(ctx context.Context, idValue string, input ReviseStockAdjustmentInput) (map[string]any, error) {
    id, err := parseUUID(idValue); if err != nil { return nil, err }
    submittedBy, err := parseUUID(input.SubmittedBy); if err != nil { return nil, fmt.Errorf("%w: submitted_by is required", ErrInvalidInput) }
    physicalQty, err := parseNumeric(input.PhysicalQty); if err != nil || !numericNonNegative(physicalQty) { return nil, fmt.Errorf("%w: physical_qty must be non-negative numeric", ErrInvalidInput) }
    reason := strings.TrimSpace(input.Reason); if reason == "" { return nil, fmt.Errorf("%w: reason is required", ErrInvalidInput) }

    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        current, err := q.LockStockAdjustment(ctx, id); if err != nil { return err }
        if current.Status != db.StockAdjustmentStatusDRAFT && current.Status != db.StockAdjustmentStatusNEEDSREVISION { return fmt.Errorf("%w: adjustment can only be revised from DRAFT or NEEDS_REVISION", ErrInvalidTransition) }
        item, err := q.UpdateStockOpnameItemPhysicalQty(ctx, db.UpdateStockOpnameItemPhysicalQtyParams{ID: current.StockOpnameItemID, PhysicalQty: physicalQty}); if err != nil { return err }
        sign, err := numericSign(item.DifferenceQty); if err != nil { return err }
        if sign == 0 { return fmt.Errorf("%w: revised physical quantity has no difference; adjustment cannot become zero", ErrInvalidInput) }
        _, err = q.UpdateStockAdjustmentForRevision(ctx, db.UpdateStockAdjustmentForRevisionParams{ID: id, AdjustmentQty: item.DifferenceQty, Reason: reason, SubmittedBy: submittedBy})
        return err
    })
    if err != nil { return nil, err }
    return s.getAdjustmentByUUID(ctx, id)
}

func (s *StockOpnameService) SubmitAdjustment(ctx context.Context, idValue, submittedByValue string) (db.StockAdjustment, error) {
    id, err := parseUUID(idValue); if err != nil { return db.StockAdjustment{}, err }
    by, err := parseUUID(submittedByValue); if err != nil { return db.StockAdjustment{}, fmt.Errorf("%w: submitted_by is required", ErrInvalidInput) }
    var result db.StockAdjustment
    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        current, err := q.LockStockAdjustment(ctx, id); if err != nil { return err }
        result, err = q.SubmitStockAdjustment(ctx, db.SubmitStockAdjustmentParams{ID: id, SubmittedBy: by}); if errors.Is(err, pgx.ErrNoRows) { return fmt.Errorf("%w: adjustment must be DRAFT", ErrInvalidTransition) }; if err != nil { return err }
        _, err = q.UpdateStockOpnameStatus(ctx, db.UpdateStockOpnameStatusParams{ID: current.StockOpnameID, Status: db.StockOpnameStatusWAITINGADJUSTMENTAPPROVAL})
        return err
    })
    return result, err
}

func (s *StockOpnameService) DecideAdjustment(ctx context.Context, idValue string, input DecideStockAdjustmentInput) (map[string]any, error) {
    id, err := parseUUID(idValue); if err != nil { return nil, err }
    approverID, err := parseUUID(input.ApproverID); if err != nil { return nil, fmt.Errorf("%w: approver_id is required", ErrInvalidInput) }
    var decision db.StockAdjustmentApprovalDecision
    switch strings.ToUpper(strings.TrimSpace(input.Decision)) { case "APPROVED": decision = db.StockAdjustmentApprovalDecisionAPPROVED; case "REJECTED": decision = db.StockAdjustmentApprovalDecisionREJECTED; default: return nil, fmt.Errorf("%w: decision must be APPROVED or REJECTED", ErrInvalidInput) }

    err = s.store.WithTx(ctx, func(q *db.Queries) error {
        adjustment, err := q.LockStockAdjustment(ctx, id); if err != nil { return err }
        if adjustment.Status != db.StockAdjustmentStatusWAITINGAPPROVAL { return fmt.Errorf("%w: adjustment must be WAITING_APPROVAL", ErrInvalidTransition) }
        user, err := q.GetUserWithRole(ctx, approverID); if err != nil { return err }
        if !user.IsActive { return fmt.Errorf("%w: approver user is inactive", ErrInvalidInput) }
        var role db.StockAdjustmentApproverRole
        switch user.RoleCode { case "CHEF": role = db.StockAdjustmentApproverRoleCHEF; case "AKUNTAN": role = db.StockAdjustmentApproverRoleAKUNTAN; default: return fmt.Errorf("%w: approver must be CHEF or AKUNTAN", ErrInvalidInput) }
        note := pgtype.Text{}; if strings.TrimSpace(input.Note) != "" { note = pgtype.Text{String: strings.TrimSpace(input.Note), Valid: true} }
        if _, err := q.CreateStockAdjustmentApproval(ctx, db.CreateStockAdjustmentApprovalParams{StockAdjustmentID: adjustment.ID, ApproverRole: role, ApproverID: approverID, EntityVersion: adjustment.Version, Status: decision, Note: note}); err != nil { return err }
        if decision == db.StockAdjustmentApprovalDecisionREJECTED { _, err = q.MarkStockAdjustmentNeedsRevision(ctx, adjustment.ID); return err }
        count, err := q.CountApprovedStockAdjustmentRolesForVersion(ctx, db.CountApprovedStockAdjustmentRolesForVersionParams{StockAdjustmentID: adjustment.ID, EntityVersion: adjustment.Version}); if err != nil { return err }
        if count < 2 { return nil }

        sign, err := numericSign(adjustment.AdjustmentQty); if err != nil { return err }
        qty := numericAbs(adjustment.AdjustmentQty)
        if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{MaterialID: adjustment.MaterialID, UnitID: adjustment.UnitID}); err != nil { return err }
        stock, err := q.LockMaterialStock(ctx, adjustment.MaterialID); if err != nil { return err }
        if stock.UnitID != adjustment.UnitID { return fmt.Errorf("%w: stock unit differs from adjustment unit", ErrConflict) }
        movementType := db.StockMovementTypeADJUSTMENTIN
        if sign > 0 {
            if _, err := q.IncreaseMaterialStock(ctx, db.IncreaseMaterialStockParams{AddQty: qty, MaterialID: adjustment.MaterialID, UnitID: adjustment.UnitID}); err != nil { return err }
        } else {
            movementType = db.StockMovementTypeADJUSTMENTOUT
            if _, err := q.DecreaseMaterialStockIfSufficient(ctx, db.DecreaseMaterialStockIfSufficientParams{SubtractQty: qty, MaterialID: adjustment.MaterialID, UnitID: adjustment.UnitID}); errors.Is(err, pgx.ErrNoRows) { return fmt.Errorf("%w: insufficient stock for adjustment", ErrInsufficientStock) } else if err != nil { return err }
        }
        if _, err := q.CreateStockMovement(ctx, db.CreateStockMovementParams{MaterialID: adjustment.MaterialID, MovementType: movementType, ReferenceType: db.StockReferenceTypeSTOCKADJUSTMENT, ReferenceID: adjustment.ID, Qty: qty, UnitID: adjustment.UnitID, MovementDate: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}, CreatedBy: approverID}); err != nil { return err }
        if _, err := q.MarkStockAdjustmentApproved(ctx, adjustment.ID); err != nil { return err }
        openCount, err := q.CountOpenStockAdjustmentsByOpname(ctx, adjustment.StockOpnameID); if err != nil { return err }
        if openCount == 0 { _, err = q.UpdateStockOpnameStatus(ctx, db.UpdateStockOpnameStatusParams{ID: adjustment.StockOpnameID, Status: db.StockOpnameStatusCOMPLETED}); return err }
        return nil
    })
    if err != nil { return nil, err }
    return s.getAdjustmentByUUID(ctx, id)
}

func (s *StockOpnameService) getOpnameByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
    opname, err := s.store.GetStockOpname(ctx, id); if err != nil { return nil, err }
    items, err := s.store.ListStockOpnameItems(ctx, id); if err != nil { return nil, err }
    adjustments, err := s.store.ListStockAdjustmentsByOpname(ctx, id); if err != nil { return nil, err }
    return map[string]any{"stock_opname": opname, "items": items, "adjustments": adjustments}, nil
}

func (s *StockOpnameService) getAdjustmentByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
    adjustment, err := s.store.GetStockAdjustment(ctx, id); if err != nil { return nil, err }
    approvals, err := s.store.ListStockAdjustmentApprovals(ctx, id); if err != nil { return nil, err }
    return map[string]any{"stock_adjustment": adjustment, "approvals": approvals}, nil
}

func (s *StockOpnameService) GetOpname(ctx context.Context, idValue string) (map[string]any, error) { id, err := parseUUID(idValue); if err != nil { return nil, err }; return s.getOpnameByUUID(ctx, id) }
func (s *StockOpnameService) GetAdjustment(ctx context.Context, idValue string) (map[string]any, error) { id, err := parseUUID(idValue); if err != nil { return nil, err }; return s.getAdjustmentByUUID(ctx, id) }
