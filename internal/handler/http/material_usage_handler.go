package http

import (
    "github.com/allifiz/go-opname-api/internal/service"
    "github.com/gofiber/fiber/v2"
)

type MaterialUsageHandler struct { service *service.MaterialUsageService }

func NewMaterialUsageHandler(service *service.MaterialUsageService) *MaterialUsageHandler { return &MaterialUsageHandler{service: service} }

func RegisterMaterialUsageRoutes(app *fiber.App, handler *MaterialUsageHandler) {
    api := app.Group("/api/v1")
    api.Post("/scheduled-menus/:id/material-usage", handler.Create)
    api.Get("/material-usages/:id", handler.Get)
    api.Post("/material-usages/:id/submit", handler.Submit)
    api.Post("/material-usages/:id/decision", handler.Decide)
}

func (h *MaterialUsageHandler) Create(c *fiber.Ctx) error {
    var input service.CreateMaterialUsageInput
    if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.Create(c.UserContext(), c.Params("id"), input)
    if err != nil { return respondError(c, err) }
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data":data})
}

func (h *MaterialUsageHandler) Get(c *fiber.Ctx) error {
    data, err := h.service.Get(c.UserContext(), c.Params("id"))
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *MaterialUsageHandler) Submit(c *fiber.Ctx) error {
    var body struct { SubmittedBy string `json:"submitted_by"` }
    if err := c.BodyParser(&body); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.Submit(c.UserContext(), c.Params("id"), body.SubmittedBy)
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}

func (h *MaterialUsageHandler) Decide(c *fiber.Ctx) error {
    var input service.DecideMaterialUsageInput
    if err := c.BodyParser(&input); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"}) }
    data, err := h.service.Decide(c.UserContext(), c.Params("id"), input)
    if err != nil { return respondError(c, err) }
    return c.JSON(fiber.Map{"data":data})
}
