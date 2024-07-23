package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start_time := time.Now()

		// Create request info attributes
		request_attrs := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("ip", c.IP()),
			slog.String("user_agent", c.Get("User-Agent")),
		}

		// Process request
		err := c.Next()

		// Log response info
		response_time := time.Since(start_time)
		status_code := c.Response().StatusCode()

		// Create response info attributes
		response_attrs := []slog.Attr{
			slog.Int("status_code", status_code),
			slog.Duration("response_time", response_time),
		}

		// Combine all attributes
		all_attrs := append(request_attrs, response_attrs...)

		log_msg := "Request processed"
		if err != nil {
			log_msg = "Request error"
			slog.LogAttrs(context.Background(), slog.LevelError, log_msg, all_attrs...)
		} else {
			slog.LogAttrs(context.Background(), slog.LevelInfo, log_msg, all_attrs...)
		}

		return err
	}
}
