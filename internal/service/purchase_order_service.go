package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInsufficientStock     = errors.New("insufficient stock")
	ErrPOCancelDeadlinePassed = errors.New("po cancel deadline passed")
)

type GeneratePurchaseOrderInput struct {
	DeliveryDate string                   `json:"delivery_date"`
	Items        []PurchaseOrderItemInput `json:"items"`
}

type PurchaseOrderItemInput struct {
	ProcurementRequestItemID string `json:"procurement_request_item_id"`
	SupplierName             string `json:"supplier_name"`
	AgreedUnitPrice          string `json:"agreed_unit_price"`
}

type CancelPurchaseOrderItemInput struct {
	CancelledBy string `json:"cancelled_by"`
}

func numericRat(n pgtype.Numeric) (*big.Rat, error) {
	if !n.Valid || n.NaN || n.InfinityModifier != pgtype.Finite || n.Int == nil {
		return nil, fmt.Errorf("invalid numeric")
	}
	r := new(big.Rat).SetInt(new(big.Int).Set(n.Int))
	if n.Exp == 0 {
		return r, nil
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs32(n.Exp))), nil)
	if n.Exp > 0 {
		r.Mul(r, new(big.Rat).SetInt(pow))
	} else {
		r.Quo(r, new(big.Rat).SetInt(pow))
	}
	return r, nil
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func numericCompare(a, b pgtype.Numeric) (int, error) {
	ar, err := numericRat(a)
	if err != nil {
		return 0, err
	}
	br, err := numericRat(b)
	if err != nil {
		return 0, err
	}
	return ar.Cmp(br), nil
}

func numericNonNegative(n pgtype.Numeric) bool {
	r, err := numericRat(n)
	return err == nil && r.Sign() >= 0
}

func generatePONumber(now time.Time) (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("PO-%s-%s", now.Format("20060102"), strings.ToUpper(hex.EncodeToString(buf))), nil
}

func jakartaNow() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Now().UTC().Add(7 * time.Hour)
	}
	return time.Now().In(loc)
}

func (s *ProcurementService) GeneratePurchaseOrder(ctx context.Context, requestIDValue string, input GeneratePurchaseOrderInput) (map[string]any, error) {
	requestID, err := parseUUID(requestIDValue)
	if err != nil {
		return nil, err
	}
	request, err := s.store.GetProcurementRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if request.Status != db.ProcurementRequestStatusVERIFIED {
		return nil, fmt.Errorf("%w: procurement request must be VERIFIED", ErrInvalidTransition)
	}

	if _, err := s.store.GetPurchaseOrderByProcurementRequest(ctx, requestID); err == nil {
		return nil, fmt.Errorf("%w: purchase order already exists for procurement request", ErrConflict)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	deliveryDate, err := parseDate(input.DeliveryDate)
	if err != nil {
		return nil, err
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("%w: purchase order items are required", ErrInvalidInput)
	}

	requestItems, err := s.store.ListProcurementRequestItems(ctx, requestID)
	if err != nil {
		return nil, err
	}
	inputByID := make(map[string]PurchaseOrderItemInput, len(input.Items))
	for _, item := range input.Items {
		id, err := parseUUID(item.ProcurementRequestItemID)
		if err != nil {
			return nil, err
		}
		key := id.String()
		if _, exists := inputByID[key]; exists {
			return nil, fmt.Errorf("%w: duplicate procurement_request_item_id", ErrInvalidInput)
		}
		item.SupplierName = strings.TrimSpace(item.SupplierName)
		if item.SupplierName == "" {
			return nil, fmt.Errorf("%w: supplier_name is required", ErrInvalidInput)
		}
		price, err := parseNumeric(item.AgreedUnitPrice)
		if err != nil || !numericNonNegative(price) {
			return nil, fmt.Errorf("%w: agreed_unit_price must be non-negative numeric", ErrInvalidInput)
		}
		inputByID[key] = item
	}

	positiveCount := 0
	for _, item := range requestItems {
		if !numericPositive(item.NetProcurementQty) {
			continue
		}
		positiveCount++
		if _, ok := inputByID[item.ID.String()]; !ok {
			return nil, fmt.Errorf("%w: every positive net procurement item must be supplied", ErrInvalidInput)
		}
	}
	if positiveCount == 0 {
		return nil, fmt.Errorf("%w: procurement request has no quantity to purchase", ErrInvalidTransition)
	}
	if len(inputByID) != positiveCount {
		return nil, fmt.Errorf("%w: input contains item not eligible for purchase", ErrInvalidInput)
	}

	poNumber, err := generatePONumber(jakartaNow())
	if err != nil {
		return nil, err
	}

	var po db.PurchaseOrder
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		var err error
		po, err = q.CreatePurchaseOrder(ctx, db.CreatePurchaseOrderParams{
			ProcurementRequestID: request.ID,
			ScheduledMenuID:      request.ScheduledMenuID,
			PoNumber:             poNumber,
			DeliveryDate:         deliveryDate,
			Status:               db.PurchaseOrderStatusVERIFIED,
		})
		if err != nil {
			return err
		}

		for _, requestItem := range requestItems {
			if !numericPositive(requestItem.NetProcurementQty) {
				continue
			}
			itemInput := inputByID[requestItem.ID.String()]
			price, err := parseNumeric(itemInput.AgreedUnitPrice)
			if err != nil {
				return err
			}
			if _, err := q.CreatePurchaseOrderItem(ctx, db.CreatePurchaseOrderItemParams{
				PurchaseOrderID:          po.ID,
				ProcurementRequestItemID: requestItem.ID,
				MaterialID:               requestItem.MaterialID,
				OrderedQty:               requestItem.NetProcurementQty,
				UnitID:                   requestItem.UnitID,
				AgreedUnitPrice:          price,
				SupplierName:             itemInput.SupplierName,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getPurchaseOrderByUUID(ctx, po.ID)
}

func (s *ProcurementService) getPurchaseOrderByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
	po, err := s.store.GetPurchaseOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListPurchaseOrderItems(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"purchase_order": po, "items": items}, nil
}

func (s *ProcurementService) GetPurchaseOrder(ctx context.Context, idValue string) (map[string]any, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return nil, err
	}
	return s.getPurchaseOrderByUUID(ctx, id)
}

func (s *ProcurementService) ListPurchaseOrdersByScheduledMenu(ctx context.Context, scheduledMenuIDValue string) ([]db.PurchaseOrder, error) {
	id, err := parseUUID(scheduledMenuIDValue)
	if err != nil {
		return nil, err
	}
	return s.store.ListPurchaseOrdersByScheduledMenu(ctx, id)
}

func (s *ProcurementService) CancelPurchaseOrderItemH1(ctx context.Context, itemIDValue string, input CancelPurchaseOrderItemInput) (db.PurchaseOrderItem, error) {
	itemID, err := parseUUID(itemIDValue)
	if err != nil {
		return db.PurchaseOrderItem{}, err
	}
	cancelledBy, err := parseUUID(input.CancelledBy)
	if err != nil {
		return db.PurchaseOrderItem{}, fmt.Errorf("%w: cancelled_by is required until authentication is implemented", ErrInvalidInput)
	}

	var cancelled db.PurchaseOrderItem
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		item, err := q.LockPurchaseOrderItemForCancellation(ctx, itemID)
		if err != nil {
			return err
		}
		if item.Status != db.PurchaseOrderItemStatusWAITING {
			return fmt.Errorf("%w: purchase order item must be WAITING", ErrInvalidTransition)
		}

		now := jakartaNow()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		delivery := time.Date(item.DeliveryDate.Time.Year(), item.DeliveryDate.Time.Month(), item.DeliveryDate.Time.Day(), 0, 0, 0, 0, now.Location())
		if !today.Before(delivery) {
			return fmt.Errorf("%w: cancellation is only allowed no later than H-1", ErrPOCancelDeadlinePassed)
		}

		stock, err := q.LockMaterialStock(ctx, item.MaterialID)
		if err != nil {
			return err
		}
		if stock.UnitID != item.UnitID {
			return fmt.Errorf("%w: stock unit differs from purchase order item unit", ErrConflict)
		}
		available, err := q.GetUnreservedStockByMaterial(ctx, item.MaterialID)
		if err != nil {
			return err
		}
		cmp, err := numericCompare(available, item.OrderedQty)
		if err != nil {
			return err
		}
		if cmp < 0 {
			return fmt.Errorf("%w: unreserved stock is not sufficient to replace ordered quantity", ErrInsufficientStock)
		}

		if _, err := q.CreateStockReservation(ctx, db.CreateStockReservationParams{
			ScheduledMenuID:          item.ScheduledMenuID,
			ProcurementRequestItemID: item.ProcurementRequestItemID,
			MaterialID:               item.MaterialID,
			Qty:                      item.OrderedQty,
			UnitID:                   item.UnitID,
		}); err != nil {
			return err
		}

		cancelled, err = q.CancelPurchaseOrderItem(ctx, db.CancelPurchaseOrderItemParams{
			ID:           item.ID,
			CancelledBy:  cancelledBy,
			CancelReason: pgtype.Text{String: "EXISTING_STOCK_SUFFICIENT", Valid: true},
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: purchase order item status changed concurrently", ErrConflict)
		}
		return err
	})
	return cancelled, err
}
