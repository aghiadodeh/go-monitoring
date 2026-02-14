package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Guard returns a Fiber middleware that validates a Bearer JWT token.
// When authRequired is false the guard is a no-op.
// When apisEnabled is false every request gets a 404.
func Guard(authRequired, apisEnabled bool, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !apisEnabled {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"statusCode": fiber.StatusNotFound,
				"message":    "Not Found",
			})
		}

		if !authRequired {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"statusCode": fiber.StatusUnauthorized,
				"message":    "unauthorized",
				"success":    false,
			})
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"statusCode": fiber.StatusUnauthorized,
				"message":    "unauthorized",
				"success":    false,
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"statusCode": fiber.StatusUnauthorized,
				"message":    "unauthorized",
				"success":    false,
			})
		}

		c.Locals("monitoring_user", claims)
		return c.Next()
	}
}
