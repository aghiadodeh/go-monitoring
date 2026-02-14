package handlers

import (
	"github.com/aghiad-odeh/go-monitoring/dto"
	"github.com/aghiad-odeh/go-monitoring/services"
	"github.com/gofiber/fiber/v2"
)

// JobHandler exposes REST endpoints for job logs.
type JobHandler struct {
	Service *services.JobService
}

// FindAll handles GET /jobs
func (h *JobHandler) FindAll(c *fiber.Ctx) error {
	var f dto.JobFilter
	if err := c.QueryParser(&f); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid query parameters"})
	}
	result, err := h.Service.FindAll(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(result)
}

// FindByID handles GET /jobs/:id
func (h *JobHandler) FindByID(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.Service.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "not found"})
	}
	return c.JSON(result)
}

// ClearAll handles DELETE /clear
func (h *JobHandler) ClearAll(c *fiber.Ctx) error {
	if err := h.Service.ClearAll(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "all monitoring data cleared"})
}
