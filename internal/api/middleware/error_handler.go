package middleware

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Check if it's a Fiber error
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return c.Status(fiberErr.Code).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "HTTP_ERROR",
					"message": fiberErr.Message,
				},
			})
		}

		// Check if it's our AppError
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			// Log internal errors
			if appErr.StatusCode >= 500 {
				logger.Error("internal error",
					slog.String("code", appErr.Code),
					slog.String("message", appErr.Message),
					slog.Any("error", appErr.Err),
				)
			}

			return c.Status(appErr.StatusCode).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    appErr.Code,
					"message": appErr.Message,
				},
			})
		}

		// Unknown error - log and return generic message
		logger.Error("unhandled error",
			slog.Any("error", err),
			slog.String("path", c.Path()),
		)

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "An unexpected error occurred",
			},
		})
	}
}
