package http

import (
	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type ProcurementHandler struct {
	service *service.ProcurementService
}

func NewProcurementHandler(service *service.ProcurementService) *ProcurementHandler {
	return &ProcurementHandler{service: service}
}

func RegisterProcurementRoutes(app *fiber.App, handler *ProcurementHandler) {
	api := app.Group("/api/v1")

	api.Post("/scheduled-menus/:id/procurement-stock-check", handler.CreateStockCheck)
	api.Get("/scheduled-menus/:id/procurement-requests", handler.ListByScheduledMenu)
	api.Get("/procurement-requests/:id", handler.GetProcurementRequest)
	api.Post("/procurement-requests/:id/submit", handler.Submit)
	api.Post("/procurement-requests/:id/verify", handler.Verify)
	api.Post("/procurement-requests/:id/reject", handler.Reject)

	api.Post("/procurement-requests/:id/purchase-order", handler.GeneratePurchaseOrder)
	api.Get("/purchase-orders/:id", handler.GetPurchaseOrder)
	api.Get("/scheduled-menus/:id/purchase-orders", handler.ListPurchaseOrdersByScheduledMenu)
	api.Post("/purchase-order-items/:id/cancel-h1", handler.CancelPurchaseOrderItemH1)
}

func (h *ProcurementHandler) CreateStockCheck(c *fiber.Ctx) error {
	data, err := h.service.CreateStockCheck(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) ListByScheduledMenu(c *fiber.Ctx) error {
	data, err := h.service.ListByScheduledMenu(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) GetProcurementRequest(c *fiber.Ctx) error {
	data, err := h.service.GetProcurementRequest(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) Submit(c *fiber.Ctx) error {
	data, err := h.service.Submit(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) Verify(c *fiber.Ctx) error {
	var input service.VerifyProcurementInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.Verify(c.UserContext(), c.Params("id"), input)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) Reject(c *fiber.Ctx) error {
	data, err := h.service.Reject(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) GeneratePurchaseOrder(c *fiber.Ctx) error {
	var input service.GeneratePurchaseOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.GeneratePurchaseOrder(c.UserContext(), c.Params("id"), input)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) GetPurchaseOrder(c *fiber.Ctx) error {
	data, err := h.service.GetPurchaseOrder(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) ListPurchaseOrdersByScheduledMenu(c *fiber.Ctx) error {
	data, err := h.service.ListPurchaseOrdersByScheduledMenu(c.UserContext(), c.Params("id"))
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}

func (h *ProcurementHandler) CancelPurchaseOrderItemH1(c *fiber.Ctx) error {
	var input service.CancelPurchaseOrderItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CancelPurchaseOrderItemH1(c.UserContext(), c.Params("id"), input)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data": data})
}
