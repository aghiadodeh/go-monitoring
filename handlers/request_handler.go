package handlers

import (
	"github.com/aghiad-odeh/go-monitoring/dto"
	"github.com/aghiad-odeh/go-monitoring/services"
	"github.com/gofiber/fiber/v2"
)

// RequestHandler exposes REST endpoints for request logs.
type RequestHandler struct {
	Service *services.RequestService
}

// FindAll handles GET /requests
func (h *RequestHandler) FindAll(c *fiber.Ctx) error {
	var f dto.RequestFilter
	if err := c.QueryParser(&f); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid query parameters"})
	}
	result, err := h.Service.FindAll(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}

// Analyze handles GET /requests/analyze
func (h *RequestHandler) Analyze(c *fiber.Ctx) error {
	var f dto.BaseFilter
	if err := c.QueryParser(&f); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid query parameters"})
	}
	result, err := h.Service.Analyze(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}

// FindByID handles GET /requests/view/:id
func (h *RequestHandler) FindByID(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.Service.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "not found"})
	}
	return c.JSON(result)
}
