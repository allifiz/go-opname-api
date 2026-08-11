package http

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allifiz/go-opname-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

func TestAuthMiddlewareAndRoleGuard(t *testing.T) {
	secret := "01234567890123456789012345678901-test-secret"
	app := fiber.New()
	app.Use("/api/v1", AuthRequired(secret))
	app.Get("/api/v1/accountant-only", RequireRoles("AKUNTAN"), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	request := httptest.NewRequest("GET", "/api/v1/accountant-only", nil)
	response, err := app.Test(request)
	if err != nil { t.Fatal(err) }
	if response.StatusCode != fiber.StatusUnauthorized { t.Fatalf("expected 401, got %d", response.StatusCode) }

	chefToken, err := service.SignAuthToken(service.AuthClaims{UserID:"chef-id", Role:"CHEF", Email:"chef@test.local", Exp:time.Now().Add(time.Hour).Unix()}, []byte(secret))
	if err != nil { t.Fatal(err) }
	request = httptest.NewRequest("GET", "/api/v1/accountant-only", nil)
	request.Header.Set("Authorization", "Bearer "+chefToken)
	response, err = app.Test(request)
	if err != nil { t.Fatal(err) }
	if response.StatusCode != fiber.StatusForbidden { t.Fatalf("expected 403, got %d", response.StatusCode) }

	accountantToken, err := service.SignAuthToken(service.AuthClaims{UserID:"accountant-id", Role:"AKUNTAN", Email:"accountant@test.local", Exp:time.Now().Add(time.Hour).Unix()}, []byte(secret))
	if err != nil { t.Fatal(err) }
	request = httptest.NewRequest("GET", "/api/v1/accountant-only", nil)
	request.Header.Set("Authorization", "Bearer "+accountantToken)
	response, err = app.Test(request)
	if err != nil { t.Fatal(err) }
	if response.StatusCode != fiber.StatusOK { t.Fatalf("expected 200, got %d", response.StatusCode) }
}
