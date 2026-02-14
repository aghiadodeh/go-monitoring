package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// LoginHandler returns a Fiber handler for POST /api/monitoring/authentication/login.
func LoginHandler(username, password, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"statusCode": fiber.StatusBadRequest,
				"message":    "invalid request body",
				"success":    false,
			})
		}

		if body.Username != username || body.Password != password {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"statusCode": fiber.StatusBadRequest,
				"message":    "wrong_credentials",
				"success":    false,
			})
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id":  body.Username + "-" + time.Now().Format(time.RFC3339),
			"exp": time.Now().Add(10 * time.Hour).Unix(),
		})
		signed, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"statusCode": fiber.StatusInternalServerError,
				"message":    "failed to generate token",
				"success":    false,
			})
		}

		return c.JSON(fiber.Map{
			"token":   signed,
			"success": true,
		})
	}
}
