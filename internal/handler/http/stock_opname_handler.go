package http

import (
    "github.com/allifiz/go-opname-api/internal/service"
    "github.com/gofiber/fiber/v2"
)

type StockOpnameHandler struct { service *service.StockOpnameService }

func NewStockOpnameHandler(service *service.StockOpnameService) *StockOpnameHandler { return &StockOpnameHandler{service: service} }

func RegisterStockOpnameRoutes(app *fiber.App, handler *StockOpnameHandler) {
    api := app.Group("/api/v1")
    api.Post("/scheduled-menus/:id/stock-opname", handler.Create)
    api.Get("/stock-opnames/:id", handler.GetOpname)
    api.Get("/stock-adjustments/:id", handler.GetAdjustment)
    api.Put("/stock-adjustments/:id", handler.ReviseAdjustment)
    api.Post("/stock-adjustments/:id/submit", handler.SubmitAdjustment)
    api.Post("/stock-adjustments/:id/decision", handler.DecideAdjustment)
}

func (h *StockOpnameHandler) Create(c *fiber.Ctx) error {
    var input service.CreateStockOpnameInput
    if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.Create(c.UserContext(), c.Params("id"), input)
    if err != nil { return respondError(c, err) }
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data":data})
}

func (h *StockOpnameHandler) GetOpname(c *fiber.Ctx) error {
    data, err := h.service.GetOpname(c.UserContext(), c.Params("id"))
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *StockOpnameHandler) GetAdjustment(c *fiber.Ctx) error {
    data, err := h.service.GetAdjustment(c.UserContext(), c.Params("id"))
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *StockOpnameHandler) ReviseAdjustment(c *fiber.Ctx) error {
    var input service.ReviseStockAdjustmentInput
    if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.ReviseAdjustment(c.UserContext(), c.Params("id"), input)
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *StockOpnameHandler) SubmitAdjustment(c *fiber.Ctx) error {
    var body struct { SubmittedBy string `json:"submitted_by"` }
    if err := c.BodyParser(&body); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.SubmitAdjustment(c.UserContext(), c.Params("id"), body.SubmittedBy)
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *StockOpnameHandler) DecideAdjustment(c *fiber.Ctx) error {
    var input service.DecideStockAdjustmentInput
    if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.DecideAdjustment(c.UserContext(), c.Params("id"), input)
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}
