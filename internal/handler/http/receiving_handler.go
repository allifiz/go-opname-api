package http

import (
	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type ReceivingHandler struct {
	service *service.ReceivingService
}

func NewReceivingHandler(service *service.ReceivingService) *ReceivingHandler {
	return &ReceivingHandler{service: service}
}

func RegisterReceivingRoutes(app *fiber.App, handler *ReceivingHandler) {
	api := app.Group("/api/v1")

	api.Post("/purchase-orders/:id/receipts", handler.CreateReceipt)
	api.Get("/purchase-orders/:id/receipts", handler.ListByPurchaseOrder)
	api.Get("/receipts/:id", handler.GetReceipt)
}

func (h *ReceivingHandler) CreateReceipt(c *fiber.Ctx) error {
	var input service.CreateReceiptInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CreateReceipt(c.UserContext(), c.Params("id"), input)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *ReceivingHandler) ListByPurchaseOrder(c *fiber.Ctx) error {
	data, err := h.service.ListByPurchaseOrder(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ReceivingHandler) GetReceipt(c *fiber.Ctx) error {
	data, err := h.service.GetReceipt(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}
