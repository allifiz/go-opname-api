package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/allifiz/go-opname-api/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

type ReceivingService struct {
	store *repository.Store
}

func NewReceivingService(store *repository.Store) *ReceivingService {
	return &ReceivingService{store: store}
}

type CreateReceiptInput struct {
	ReceivedBy string                 `json:"received_by"`
	Note       string                 `json:"note"`
	Items      []CreateReceiptItemInput `json:"items"`
	Documents  []CreateReceiptDocumentInput `json:"documents"`
}

type CreateReceiptItemInput struct {
	PurchaseOrderItemID string `json:"purchase_order_item_id"`
	ReceivedQty         string `json:"received_qty"`
}

type CreateReceiptDocumentInput struct {
	DocumentType string `json:"document_type"`
	FileURL      string `json:"file_url"`
	FileName     string `json:"file_name"`
}

func parseReceiptDocumentType(value string) (db.ReceiptDocumentType, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "INVOICE":
		return db.ReceiptDocumentTypeINVOICE, nil
	case "NOTA":
		return db.ReceiptDocumentTypeNOTA, nil
	case "PHOTO":
		return db.ReceiptDocumentTypePHOTO, nil
	case "OTHER":
		return db.ReceiptDocumentTypeOTHER, nil
	default:
		return "", fmt.Errorf("%w: invalid document_type", ErrInvalidInput)
	}
}

func (s *ReceivingService) CreateReceipt(ctx context.Context, purchaseOrderIDValue string, input CreateReceiptInput) (map[string]any, error) {
	purchaseOrderID, err := parseUUID(purchaseOrderIDValue)
	if err != nil {
		return nil, err
	}
	receivedBy, err := parseUUID(input.ReceivedBy)
	if err != nil {
		return nil, fmt.Errorf("%w: received_by is required until authentication is implemented", ErrInvalidInput)
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("%w: receipt items are required", ErrInvalidInput)
	}

	if _, err := s.store.GetPurchaseOrder(ctx, purchaseOrderID); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(input.Items))
	parsedQty := make(map[string]pgtype.Numeric, len(input.Items))
	for _, itemInput := range input.Items {
		itemID, err := parseUUID(itemInput.PurchaseOrderItemID)
		if err != nil {
			return nil, err
		}
		key := itemID.String()
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: duplicate purchase_order_item_id", ErrInvalidInput)
		}
		seen[key] = struct{}{}

		qty, err := parseNumeric(itemInput.ReceivedQty)
		if err != nil || !numericNonNegative(qty) {
			return nil, fmt.Errorf("%w: received_qty must be non-negative numeric", ErrInvalidInput)
		}
		parsedQty[key] = qty
	}

	for _, doc := range input.Documents {
		if _, err := parseReceiptDocumentType(doc.DocumentType); err != nil {
			return nil, err
		}
		if strings.TrimSpace(doc.FileURL) == "" || strings.TrimSpace(doc.FileName) == "" {
			return nil, fmt.Errorf("%w: document file_url and file_name are required", ErrInvalidInput)
		}
	}

	var receipt db.Receipt
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		var note pgtype.Text
		if strings.TrimSpace(input.Note) != "" {
			note = pgtype.Text{String: strings.TrimSpace(input.Note), Valid: true}
		}

		var err error
		receipt, err = q.CreateReceipt(ctx, db.CreateReceiptParams{
			PurchaseOrderID: purchaseOrderID,
			ReceivedBy:      receivedBy,
			Note:            note,
		})
		if err != nil {
			return err
		}

		for _, itemInput := range input.Items {
			itemID, _ := parseUUID(itemInput.PurchaseOrderItemID)
			qty := parsedQty[itemID.String()]

			poItem, err := q.LockPurchaseOrderItemForReceipt(ctx, itemID)
			if err != nil {
				return err
			}
			if poItem.PurchaseOrderID != purchaseOrderID {
				return fmt.Errorf("%w: purchase order item does not belong to purchase order", ErrInvalidInput)
			}
			if poItem.Status == db.PurchaseOrderItemStatusCANCELLED ||
				poItem.Status == db.PurchaseOrderItemStatusOVERRECEIVED ||
				poItem.Status == db.PurchaseOrderItemStatusFULFILLED {
				return fmt.Errorf("%w: purchase order item cannot receive more goods", ErrInvalidTransition)
			}

			receiptItem, err := q.CreateReceiptItem(ctx, db.CreateReceiptItemParams{
				ReceiptID:           receipt.ID,
				PurchaseOrderItemID: poItem.ID,
				MaterialID:          poItem.MaterialID,
				ReceivedQty:         qty,
				UnitID:              poItem.UnitID,
				AgreedUnitPrice:     poItem.AgreedUnitPrice,
			})
			if err != nil {
				return err
			}

			if numericPositive(qty) {
				if err := q.EnsureMaterialStock(ctx, db.EnsureMaterialStockParams{
					MaterialID: poItem.MaterialID,
					UnitID:     poItem.UnitID,
				}); err != nil {
					return err
				}
				stock, err := q.LockMaterialStock(ctx, poItem.MaterialID)
				if err != nil {
					return err
				}
				if stock.UnitID != poItem.UnitID {
					return fmt.Errorf("%w: stock unit differs from received item unit", ErrConflict)
				}
				if _, err := q.IncreaseMaterialStock(ctx, db.IncreaseMaterialStockParams{
					AddQty:     qty,
					MaterialID: poItem.MaterialID,
					UnitID:     poItem.UnitID,
				}); err != nil {
					return err
				}
				if _, err := q.CreateStockMovement(ctx, db.CreateStockMovementParams{
					MaterialID:    poItem.MaterialID,
					MovementType:  db.StockMovementTypeIN,
					ReferenceType: db.StockReferenceTypePORECEIPT,
					ReferenceID:   receiptItem.ID,
					Qty:           qty,
					UnitID:        poItem.UnitID,
					MovementDate:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
					CreatedBy:     receivedBy,
				}); err != nil {
					return err
				}
			}

			totalReceived, err := q.SumReceivedQtyByPurchaseOrderItem(ctx, poItem.ID)
			if err != nil {
				return err
			}
			cmp, err := numericCompare(totalReceived, poItem.OrderedQty)
			if err != nil {
				return err
			}

			status := db.PurchaseOrderItemStatusNOTRECEIVED
			if numericPositive(totalReceived) {
				switch {
				case cmp < 0:
					status = db.PurchaseOrderItemStatusPARTIALRECEIVED
				case cmp == 0:
					status = db.PurchaseOrderItemStatusRECEIVED
				default:
					status = db.PurchaseOrderItemStatusOVERRECEIVED
				}
			}
			if _, err := q.UpdatePurchaseOrderItemReceiptStatus(ctx, db.UpdatePurchaseOrderItemReceiptStatusParams{
				ID:     poItem.ID,
				Status: status,
			}); err != nil {
				return err
			}
		}

		for _, docInput := range input.Documents {
			documentType, _ := parseReceiptDocumentType(docInput.DocumentType)
			if _, err := q.CreateReceiptDocument(ctx, db.CreateReceiptDocumentParams{
				ReceiptID:    receipt.ID,
				DocumentType: documentType,
				FileUrl:      strings.TrimSpace(docInput.FileURL),
				FileName:     strings.TrimSpace(docInput.FileName),
				UploadedBy:   receivedBy,
			}); err != nil {
				return err
			}
		}

		poStatus, err := q.CalculatePurchaseOrderReceiptStatus(ctx, purchaseOrderID)
		if err != nil {
			return err
		}
		if _, err := q.UpdatePurchaseOrderStatus(ctx, db.UpdatePurchaseOrderStatusParams{
			ID:     purchaseOrderID,
			Status: poStatus,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.getReceiptByUUID(ctx, receipt.ID)
}

func (s *ReceivingService) getReceiptByUUID(ctx context.Context, id pgtype.UUID) (map[string]any, error) {
	receipt, err := s.store.GetReceipt(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListReceiptItems(ctx, id)
	if err != nil {
		return nil, err
	}
	documents, err := s.store.ListReceiptDocuments(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"receipt":   receipt,
		"items":     items,
		"documents": documents,
	}, nil
}

func (s *ReceivingService) GetReceipt(ctx context.Context, idValue string) (map[string]any, error) {
	id, err := parseUUID(idValue)
	if err != nil {
		return nil, err
	}
	return s.getReceiptByUUID(ctx, id)
}

func (s *ReceivingService) ListByPurchaseOrder(ctx context.Context, purchaseOrderIDValue string) ([]db.Receipt, error) {
	id, err := parseUUID(purchaseOrderIDValue)
	if err != nil {
		return nil, err
	}
	return s.store.ListReceiptsByPurchaseOrder(ctx, id)
}
