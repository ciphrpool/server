package middleware

import (
	"errors"
	"os"

	"github.com/gofiber/fiber/v2"
)

func WithKey(key string, real_key func() (string, error)) fiber.Handler {
	if real_key == nil {
		real_key = func() (string, error) {
			result := os.Getenv(key)
			if result == "" {
				return "", errors.New("key not found")
			}
			return result, nil
		}
	}
	return func(c *fiber.Ctx) error {
		apiKey := c.Get(key)
		correct_key, err := real_key()
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "cannont check API key",
			})
		}
		if apiKey != correct_key {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}
		return c.Next()
	}
}
