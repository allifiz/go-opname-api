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

var ErrShortageQtyExceeded = errors.New("shortage quantity exceeded")

type DirectPurchaseService struct {
	store *repository.Store
}

func NewDirectPurchaseService(store *repository.Store) *DirectPurchaseService {
	return &DirectPurchaseService{store: store}
}

type CreateShortageDirectPurchaseInput struct {
	Qty         string `json:"qty"`
	UnitPrice   string `json:"unit_price"`
	SourceName  string `json:"source_name"`
	PurchasedBy string `json:"purchased_by"`
	Note        string `json:"note"`
}

type AdditionalRequirementPriceInput struct {
	MaterialID string `json:"material_id"`
	UnitPrice  string `json:"unit_price"`
}

type CreateAdditionalRequirementDirectPurchaseInput struct {
	NewPortions int32                             `json:"new_portions"`
	SourceName  string                            `json:"source_name"`
	PurchasedBy string                            `json:"purchased_by"`
	Note        string                            `json:"note"`
	Prices      []AdditionalRequirementPriceInput `json:"prices"`
}

func nullableText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func (s *DirectPurchaseService) addStockIn(
	ctx context.Context,
	q *db.Queries,
	materialID pgtype.UUID,
	unitID pgtype.UUID,
	qty pgtype.Numeric,
	referenceType db.StockReferenceType,
	referenceID pgtype.UUID,
	createdBy pgtype.UUID,
) error {
	if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{
		MaterialID: materialID,
		UnitID:     unitID,
	}); err != nil {
		return err
	}
	stock, err := q.LockMaterialStock(ctx, materialID)
	if err != nil {
		return err
	}
	if stock.UnitID != unitID {
		return fmt.Errorf("%w: stock unit differs from direct purchase item unit", ErrConflict)
	}
	if _, err := q.IncreaseMaterialStock(ctx, db.IncreaseMaterialStockParams{
		AddQty:     qty,
		MaterialID: materialID,
		UnitID:     unitID,
	}); err != nil {
		return err
	}
	_, err = q.CreateStockMovement(ctx, db.CreateStockMovementParams{
		MaterialID:    materialID,
		MovementType:  db.StockMovementTypeIN,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Qty:           qty,
		UnitID:        unitID,
		MovementDate:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		CreatedBy:     createdBy,
	})
	return err
}

func (s *DirectPurchaseService) CreateShortage(
	ctx context.Context,
	purchaseOrderItemIDValue string,
	input CreateShortageDirectPurchaseInput,
) (map[string]any, error) {
	poItemID, err := parseUUID(purchaseOrderItemIDValue)
	if err != nil {
		return nil, err
	}
	purchasedBy, err := parseUUID(input.PurchasedBy)
	if err != nil {
		return nil, fmt.Errorf("%w: purchased_by is required until authentication is implemented", ErrInvalidInput)
	}
	input.SourceName = strings.TrimSpace(input.SourceName)
	if input.SourceName == "" {
		return nil, fmt.Errorf("%w: source_name is required", ErrInvalidInput)
	}
	qty, err := parseNumeric(input.Qty)
	if err != nil || !numericPositive(qty) {
		return nil, fmt.Errorf("%w: qty must be positive numeric", ErrInvalidInput)
	}
	unitPrice, err := parseNumeric(input.UnitPrice)
	if err != nil || !numericNonNegative(unitPrice) {
		return nil, fmt.Errorf("%w: unit_price must be non-negative numeric", ErrInvalidInput)
	}

	var purchase db.DirectPurchase
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		poItem, err := q.LockPurchaseOrderItemForShortagePurchase(ctx, poItemID)
		if err != nil {
			return err
		}
		if poItem.Status == db.PurchaseOrderItemStatusCANCELLED ||
			poItem.Status == db.PurchaseOrderItemStatusOVERRECEIVED ||
			poItem.Status == db.PurchaseOrderItemStatusFULFILLED {
			return fmt.Errorf("%w: purchase order item is not eligible for shortage purchase", ErrInvalidTransition)
		}

		remaining, err := q.GetRemainingShortageByPOItem(ctx, poItem.ID)
		if err != nil {
			return err
		}
		if !numericPositive(remaining) {
			return fmt.Errorf("%w: purchase order item has no remaining shortage", ErrInvalidTransition)
		}
		cmp, err := numericCompare(qty, remaining)
		if err != nil {
			return err
		}
		if cmp > 0 {
			return fmt.Errorf("%w: requested shortage purchase exceeds remaining shortage", ErrShortageQtyExceeded)
		}

		purchase, err = q.CreateDirectPurchase(ctx, db.CreateDirectPurchaseParams{
			ScheduledMenuID: poItem.ScheduledMenuID,
			PurchaseType:    db.DirectPurchaseTypeSHORTAGE,
			SourceName:      input.SourceName,
			PurchasedBy:     purchasedBy,
			Note:            nullableText(input.Note),
		})
		if err != nil {
			return err
		}
		purchaseItem, err := q.CreateDirectPurchaseItem(ctx, db.CreateDirectPurchaseItemParams{
			DirectPurchaseID:   purchase.ID,
			PurchaseOrderItemID: poItem.ID,
			MaterialID:         poItem.MaterialID,
			Qty:                qty,
			UnitID:             poItem.UnitID,
			UnitPrice:          unitPrice,
		})
		if err != nil {
			return err
		}
		if err := s.addStockIn(ctx, q, poItem.MaterialID, poItem.UnitID, qty, db.StockReferenceTypeSHORTAGEPURCHASE, purchaseItem.ID, purchasedBy); err != nil {
			return err
		}

		if cmp == 0 {
			if _, err := q.UpdatePurchaseOrderItemReceiptStatus(ctx, db.UpdatePurchaseOrderItemReceiptStatusParams{
				ID:     poItem.ID,
				Status: db.PurchaseOrderItemStatusFULFILLED,
			}); err != nil {
				return err
			}
			poStatus, err := q.CalculatePurchaseOrderReceiptStatus(ctx, poItem.PurchaseOrderID)
			if err != nil {
				return err
			}
			if _, err := q.UpdatePurchaseOrderStatus(ctx, db.UpdatePurchaseOrderStatusParams{ID: poItem.PurchaseOrderID, Status: poStatus}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getDirectPurchaseByUUID(ctx, purchase.ID)
}

func (s *DirectPurchaseService) CreateAdditionalRequirement(
	ctx context.Context,
	scheduledMenuIDValue string,
	input CreateAdditionalRequirementDirectPurchaseInput,
) (map[string]any, error) {
	scheduledMenuID, err := parseUUID(scheduledMenuIDValue)
	if err != nil {
		return nil, err
	}
	purchasedBy, err := parseUUID(input.PurchasedBy)
	if err != nil {
		return nil, fmt.Errorf("%w: purchased_by is required until authentication is implemented", ErrInvalidInput)
	}
	input.SourceName = strings.TrimSpace(input.SourceName)
	if input.SourceName == "" {
		return nil, fmt.Errorf("%w: source_name is required", ErrInvalidInput)
	}
	if input.NewPortions <= 0 || len(input.Prices) == 0 {
		return nil, fmt.Errorf("%w: new_portions and prices are required", ErrInvalidInput)
	}

	prices := make(map[string]pgtype.Numeric, len(input.Prices))
	for _, priceInput := range input.Prices {
		materialID, err := parseUUID(priceInput.MaterialID)
		if err != nil {
			return nil, err
		}
		key := materialID.String()
		if _, exists := prices[key]; exists {
			return nil, fmt.Errorf("%w: duplicate material_id in prices", ErrInvalidInput)
		}
		price, err := parseNumeric(priceInput.UnitPrice)
		if err != nil || !numericNonNegative(price) {
			return nil, fmt.Errorf("%w: unit_price must be non-negative numeric", ErrInvalidInput)
		}
		prices[key] = price
	}

	var purchase db.DirectPurchase
	var requirement db.AdditionalRequirement
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		scheduled, err := q.LockScheduledMenuForAdditionalRequirement(ctx, scheduledMenuID)
		if err != nil {
			return err
		}
		currentPortions := scheduled.PlannedPortions
		latest, err := q.GetLatestAdditionalRequirementByScheduledMenu(ctx, scheduledMenuID)
		if err == nil {
			currentPortions = latest.NewPortions
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if input.NewPortions <= currentPortions {
			return fmt.Errorf("%w: new_portions must be greater than current effective portions", ErrInvalidInput)
		}
		delta := input.NewPortions - currentPortions

		additionalItems, err := q.GetScheduledMenuAdditionalRequirements(ctx, db.GetScheduledMenuAdditionalRequirementsParams{
			AdditionalPortions: delta,
			ScheduledMenuID:    scheduledMenuID,
		})
		if err != nil {
			return err
		}
		if len(additionalItems) == 0 {
			return fmt.Errorf("%w: scheduled menu has no material requirements", ErrInvalidInput)
		}
		if len(prices) != len(additionalItems) {
			return fmt.Errorf("%w: every additional material must have exactly one price", ErrInvalidInput)
		}
		for _, item := range additionalItems {
			if _, ok := prices[item.MaterialID.String()]; !ok {
				return fmt.Errorf("%w: missing price for additional material", ErrInvalidInput)
			}
		}

		requirement, err = q.CreateAdditionalRequirement(ctx, db.CreateAdditionalRequirementParams{
			ScheduledMenuID: scheduledMenuID,
			PreviousPortions: currentPortions,
			NewPortions:      input.NewPortions,
			CreatedBy:        purchasedBy,
		})
		if err != nil {
			return err
		}
		purchase, err = q.CreateDirectPurchase(ctx, db.CreateDirectPurchaseParams{
			ScheduledMenuID: scheduledMenuID,
			PurchaseType:    db.DirectPurchaseTypeADDITIONALREQUIREMENT,
			SourceName:      input.SourceName,
			PurchasedBy:     purchasedBy,
			Note:            nullableText(input.Note),
		})
		if err != nil {
			return err
		}

		for _, item := range additionalItems {
			requirementItem, err := q.CreateAdditionalRequirementItem(ctx, db.CreateAdditionalRequirementItemParams{
				AdditionalRequirementID: requirement.ID,
				MaterialID:              item.MaterialID,
				AdditionalQty:            item.AdditionalQty,
				UnitID:                  item.UnitID,
			})
			if err != nil {
				return err
			}
			purchaseItem, err := q.CreateDirectPurchaseItem(ctx, db.CreateDirectPurchaseItemParams{
				DirectPurchaseID:            purchase.ID,
				AdditionalRequirementItemID: requirementItem.ID,
				MaterialID:                  item.MaterialID,
				Qty:                         item.AdditionalQty,
				UnitID:                      item.UnitID,
				UnitPrice:                   prices[item.MaterialID.String()],
			})
			if err != nil {
				return err
			}
			if err := s.addStockIn(ctx, q, item.MaterialID, item.UnitID, item.AdditionalQty, db.StockReferenceTypeADDITIONALREQUIREMENT, purchaseItem.ID, purchasedBy); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result, err := s.getDirectPurchaseByUUID(ctx, purchase.ID)
	if err != nil {
		return nil, err
	}
	result["additional_requirement"] = requirement
	return result, nil
}

func (s *DirectPurchaseService) getDirectPurchaseByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
	purchase, err := s.store.GetDirectPurchase(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListDirectPurchaseItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"direct_purchase": purchase, "items": items}, nil
}

func (s *DirectPurchaseService) Get(ctx context.Context, idValue string) (map[string]any, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return nil, err
	}
	return s.getDirectPurchaseByUUID(ctx, id)
}

func (s *DirectPurchaseService) ListByScheduledMenu(ctx context.Context, scheduledMenuIDValue string) ([]db.DirectPurchase, error) {
	id, err := parseUUID(scheduledMenuIDValue)
	if err != nil {
		return nil, err
	}
	return s.store.ListDirectPurchasesByScheduledMenu(ctx, id)
}
