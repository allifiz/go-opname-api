package http

import (
	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type DirectPurchaseHandler struct { service *service.DirectPurchaseService }
func NewDirectPurchaseHandler(service *service.DirectPurchaseService) *DirectPurchaseHandler { return &DirectPurchaseHandler{service: service} }

func RegisterDirectPurchaseRoutes(app *fiber.App, handler *DirectPurchaseHandler) {
	api := app.Group("/api/v1")
	pengawas := RequireRoles("PENGAWAS_BAHAN_BAKU")
	api.Post("/purchase-order-items/:id/direct-purchases/shortage", pengawas, handler.CreateShortage)
	api.Post("/scheduled-menus/:id/direct-purchases/additional-requirement", pengawas, handler.CreateAdditionalRequirement)
	api.Get("/direct-purchases/:id", handler.Get)
	api.Get("/scheduled-menus/:id/direct-purchases", handler.ListByScheduledMenu)
}

func (h *DirectPurchaseHandler) CreateShortage(c *fiber.Ctx) error {
	var input service.CreateShortageDirectPurchaseInput
	if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
	input.PurchasedBy = ActorID(c)
	data, err := h.service.CreateShortage(c.UserContext(), c.Params("id"), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data":data})
}

func (h *DirectPurchaseHandler) CreateAdditionalRequirement(c *fiber.Ctx) error {
	var input service.CreateAdditionalRequirementDirectPurchaseInput
	if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
	input.PurchasedBy = ActorID(c)
	data, err := h.service.CreateAdditionalRequirement(c.UserContext(), c.Params("id"), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data":data})
}

func (h *DirectPurchaseHandler) Get(c *fiber.Ctx) error { data, err := h.service.Get(c.UserContext(), c.Params("id")); if err != nil { return respondError(c, err) }; return c.JSON(fiber.Map{"data":data}) }
func (h *DirectPurchaseHandler) ListByScheduledMenu(c *fiber.Ctx) error { data, err := h.service.ListByScheduledMenu(c.UserContext(), c.Params("id")); if err != nil { return respondError(c, err) }; return c.JSON(fiber.Map{"data":data}) }
