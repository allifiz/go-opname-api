package http

import (
	"errors"

	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type CoreHandler struct {
	service *service.CoreService
}

func NewCoreHandler(service *service.CoreService) *CoreHandler {
	return &CoreHandler{service: service}
}

func RegisterCoreRoutes(app *fiber.App, handler *CoreHandler) {
	api := app.Group("/api/v1")

	api.Get("/units", handler.ListUnits)

	api.Get("/materials", handler.ListMaterials)
	api.Get("/materials/:id", handler.GetMaterial)
	api.Post("/materials", handler.CreateMaterial)
	api.Put("/materials/:id", handler.UpdateMaterial)

	api.Get("/periods", handler.ListPeriods)
	api.Post("/periods", handler.CreatePeriod)

	api.Get("/menu-templates", handler.ListMenuTemplates)
	api.Get("/menu-templates/:id", handler.GetMenuTemplate)
	api.Post("/menu-templates", handler.CreateMenuTemplate)

	api.Post("/scheduled-menus", handler.CreateScheduledMenu)
	api.Get("/scheduled-menus/:id", handler.GetScheduledMenu)
}

func respondError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "internal server error"

	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status = fiber.StatusBadRequest
		message = err.Error()
	case errors.Is(err, service.ErrInsufficientStock), errors.Is(err, service.ErrShortageQtyExceeded):
		status = fiber.StatusUnprocessableEntity
		message = err.Error()
	case errors.Is(err, service.ErrConflict), errors.Is(err, service.ErrInvalidTransition), errors.Is(err, service.ErrPOCancelDeadlinePassed):
		status = fiber.StatusConflict
		message = err.Error()
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		status = fiber.StatusConflict
		message = "resource already exists"
	case errors.Is(err, pgx.ErrNoRows):
		status = fiber.StatusNotFound
		message = "resource not found"
	}

	return c.Status(status).JSON(fiber.Map{"error": message})
}

func (h *CoreHandler) ListUnits(c *fiber.Ctx) error {
	data, err := h.service.ListUnits(c.UserContext())
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) ListMaterials(c *fiber.Ctx) error {
	data, err := h.service.ListMaterials(c.UserContext())
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) GetMaterial(c *fiber.Ctx) error {
	data, err := h.service.GetMaterial(c.UserContext(), c.Params("id"))
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) CreateMaterial(c *fiber.Ctx) error {
	var input service.CreateMaterialInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CreateMaterial(c.UserContext(), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) UpdateMaterial(c *fiber.Ctx) error {
	var input service.UpdateMaterialInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.UpdateMaterial(c.UserContext(), c.Params("id"), input)
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) ListPeriods(c *fiber.Ctx) error {
	data, err := h.service.ListPeriods(c.UserContext())
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) CreatePeriod(c *fiber.Ctx) error {
	var input service.CreatePeriodInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CreatePeriod(c.UserContext(), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) ListMenuTemplates(c *fiber.Ctx) error {
	data, err := h.service.ListMenuTemplates(c.UserContext())
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) GetMenuTemplate(c *fiber.Ctx) error {
	data, err := h.service.GetMenuTemplate(c.UserContext(), c.Params("id"))
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) CreateMenuTemplate(c *fiber.Ctx) error {
	var input service.CreateMenuTemplateInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CreateMenuTemplate(c.UserContext(), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) CreateScheduledMenu(c *fiber.Ctx) error {
	var input service.CreateScheduledMenuInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	data, err := h.service.CreateScheduledMenu(c.UserContext(), input)
	if err != nil { return respondError(c, err) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": data})
}

func (h *CoreHandler) GetScheduledMenu(c *fiber.Ctx) error {
	data, err := h.service.GetScheduledMenu(c.UserContext(), c.Params("id"))
	if err != nil { return respondError(c, err) }
	return c.JSON(fiber.Map{"data": data})
}
