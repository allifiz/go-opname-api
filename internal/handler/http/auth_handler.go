package http

import (
	"strings"

	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

const (
	localUserID = "auth_user_id"
	localRole   = "auth_role"
	localEmail  = "auth_email"
)

type AuthHandler struct { service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

func RegisterPublicAuthRoutes(app *fiber.App, handler *AuthHandler) {
	app.Post("/api/v1/auth/login", handler.Login)
}

func RegisterProtectedAuthRoutes(app *fiber.App) {
	app.Get("/api/v1/auth/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"user_id": c.Locals(localUserID),
			"role": c.Locals(localRole),
			"email": c.Locals(localEmail),
		}})
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var input service.LoginInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"invalid json body"})
	}
	data, err := h.service.Login(c.UserContext(), input)
	if err != nil {
		if err == service.ErrUnauthorized { return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"invalid credentials"}) }
		return respondError(c, err)
	}
	return c.JSON(fiber.Map{"data":data})
}

func AuthRequired(secret string) fiber.Handler {
	key := []byte(secret)
	return func(c *fiber.Ctx) error {
		header := strings.TrimSpace(c.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"missing bearer token"})
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		claims, err := service.ParseAuthToken(token, key)
		if err != nil { return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error":"invalid or expired token"}) }
		c.Locals(localUserID, claims.UserID)
		c.Locals(localRole, claims.Role)
		c.Locals(localEmail, claims.Email)
		return c.Next()
	}
}

func ActorID(c *fiber.Ctx) string {
	value, _ := c.Locals(localUserID).(string)
	return value
}

func RequireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles { allowed[role] = struct{}{} }
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(localRole).(string)
		if _, ok := allowed[role]; !ok { return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error":"forbidden for current role"}) }
		return c.Next()
	}
}
