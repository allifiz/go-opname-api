package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/allifiz/go-opname-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type ProcurementService struct {
	store *repository.Store
}

func NewProcurementService(store *repository.Store) *ProcurementService {
	return &ProcurementService{store: store}
}

type VerifyProcurementInput struct {
	VerifiedBy string `json:"verified_by"`
}

func numericPositive(n pgtype.Numeric) bool {
	return n.Valid && !n.NaN && n.Int != nil && n.Int.Sign() > 0
}

func (s *ProcurementService) CreateStockCheck(ctx context.Context, scheduledMenuIDValue string) (map[string]any, error) {
	scheduledMenuID, err := parseUUID(scheduledMenuIDValue)
	if err != nil {
		return nil, err
	}

	if _, err := s.store.GetScheduledMenu(ctx, scheduledMenuID); err != nil {
		return nil, err
	}

	existingRequests, err := s.store.ListProcurementRequestsByScheduledMenu(ctx, scheduledMenuID)
	if err != nil {
		return nil, err
	}
	if len(existingRequests) > 0 {
		return nil, fmt.Errorf("%w: procurement request already exists for scheduled menu", ErrConflict)
	}

	var request db.ProcurementRequest
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		grossRequirements, err := q.GetScheduledMenuGrossRequirements(ctx, scheduledMenuID)
		if err != nil {
			return err
		}
		if len(grossRequirements) == 0 {
			return fmt.Errorf("%w: scheduled menu has no material requirements", ErrInvalidInput)
		}

		request, err = q.CreateProcurementRequest(ctx, db.CreateProcurementRequestParams{
			ScheduledMenuID: scheduledMenuID,
			CheckedAt:       pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		})
		if err != nil {
			return err
		}

		for _, requirement := range grossRequirements {
			if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{
				MaterialID: requirement.MaterialID,
				UnitID:     requirement.UnitID,
			}); err != nil {
				return err
			}

			stock, err := q.LockMaterialStock(ctx, requirement.MaterialID)
			if err != nil {
				return err
			}
			if stock.UnitID != requirement.UnitID {
				return fmt.Errorf("%w: material stock unit differs from scheduled material unit", ErrConflict)
			}

			availability, err := q.GetProcurementStockAvailability(ctx, db.GetProcurementStockAvailabilityParams{
				GrossRequirementQty: requirement.GrossRequirementQty,
				MaterialID:          requirement.MaterialID,
			})
			if err != nil {
				return err
			}

			item, err := q.CreateProcurementRequestItem(ctx, db.CreateProcurementRequestItemParams{
				ProcurementRequestID: request.ID,
				MaterialID:           requirement.MaterialID,
				GrossRequirementQty:  requirement.GrossRequirementQty,
				ExistingStockQty:     availability.ExistingStockQty,
				ReservedStockQty:     availability.ReservedStockQty,
				NetProcurementQty:    availability.NetProcurementQty,
				UnitID:               requirement.UnitID,
			})
			if err != nil {
				return err
			}

			if numericPositive(availability.AllocationQty) {
				if _, err := q.CreateStockReservation(ctx, db.CreateStockReservationParams{
					ScheduledMenuID:          scheduledMenuID,
					ProcurementRequestItemID: item.ID,
					MaterialID:               requirement.MaterialID,
					Qty:                      availability.AllocationQty,
					UnitID:                   requirement.UnitID,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.getProcurementRequestByUUID(ctx, request.ID)
}

func (s *ProcurementService) getProcurementRequestByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
	request, err := s.store.GetProcurementRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListProcurementRequestItems(ctx, id)
	if err != nil {
		return nil, err
	}
	reservations, err := s.store.ListStockReservationsByProcurementRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"procurement_request": request,
		"items":               items,
		"reservations":        reservations,
	}, nil
}

func (s *ProcurementService) GetProcurementRequest(ctx context.Context, idValue string) (map[string]any, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return nil, err
	}
	return s.getProcurementRequestByUUID(ctx, id)
}

func (s *ProcurementService) ListByScheduledMenu(ctx context.Context, scheduledMenuIDValue string) ([]db.ProcurementRequest, error) {
	id, err := parseUUID(scheduledMenuIDValue)
	if err != nil {
		return nil, err
	}
	return s.store.ListProcurementRequestsByScheduledMenu(ctx, id)
}

func (s *ProcurementService) Submit(ctx context.Context, idValue string) (db.ProcurementRequest, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	current, err := s.store.GetProcurementRequest(ctx, id)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	if current.Status != db.ProcurementRequestStatusDRAFT && current.Status != db.ProcurementRequestStatusREJECTED {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request must be DRAFT or REJECTED", ErrInvalidTransition)
	}
	request, err := s.store.SubmitProcurementRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request status changed concurrently", ErrConflict)
	}
	return request, err
}

func (s *ProcurementService) Verify(ctx context.Context, idValue string, input VerifyProcurementInput) (db.ProcurementRequest, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	verifiedBy, err := parseUUID(input.VerifiedBy)
	if err != nil {
		return db.ProcurementRequest{}, fmt.Errorf("%w: verified_by is required until authentication is implemented", ErrInvalidInput)
	}
	current, err := s.store.GetProcurementRequest(ctx, id)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	if current.Status != db.ProcurementRequestStatusWAITINGVERIFICATION {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request must be WAITING_VERIFICATION", ErrInvalidTransition)
	}
	request, err := s.store.VerifyProcurementRequest(ctx, db.VerifyProcurementRequestParams{ID: id, VerifiedBy: verifiedBy})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request status changed concurrently", ErrConflict)
	}
	return request, err
}

func (s *ProcurementService) Reject(ctx context.Context, idValue string) (db.ProcurementRequest, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	current, err := s.store.GetProcurementRequest(ctx, id)
	if err != nil {
		return db.ProcurementRequest{}, err
	}
	if current.Status != db.ProcurementRequestStatusWAITINGVERIFICATION {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request must be WAITING_VERIFICATION", ErrInvalidTransition)
	}
	request, err := s.store.RejectProcurementRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ProcurementRequest{}, fmt.Errorf("%w: request status changed concurrently", ErrConflict)
	}
	return request, err
}
