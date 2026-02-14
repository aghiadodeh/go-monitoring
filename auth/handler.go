package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// LoginHandler returns a Fiber handler for POST /api/monitoring/authentication/login.
func LoginHandler(username, password, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Username string `json:"username" validate:"required"`
			Password string `json:"password" validate:"required"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"statusCode": fiber.StatusBadRequest,
				"message":    "invalid request body",
				"success":    false,
			})
		}

		var validate = validator.New()
		if err := validate.Struct(body); err != nil {
			// Collect error messages
			var messages []string
			for _, err := range err.(validator.ValidationErrors) {
				messages = append(messages, fmt.Sprintf("%s is %s", err.Field(), err.Tag()))
			}
			return fiber.NewError(fiber.StatusBadRequest, strings.Join(messages, ", "))
		}

		if body.Username != username || body.Password != password {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"statusCode": fiber.StatusBadRequest,
				"message":    "Wrong Credentials",
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

		return c.JSON(signed)
	}
}
