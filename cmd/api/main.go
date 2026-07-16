package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/allifiz/go-opname-api/internal/config"
	"github.com/gofiber/fiber/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := config.OpenDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New(fiber.Config{
		AppName: "Stock Opname API",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	go func() {
		<-ctx.Done()
		if err := app.Shutdown(); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("API listening on :%s", cfg.AppPort)
	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Printf("server stopped: %v", err)
	}
}
