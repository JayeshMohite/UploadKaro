package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func RequestIDLogger(c *fiber.Ctx) error {
	requestID := c.Get("X-Request-ID")

	// set it in the context
	c.Set("X-Request-ID", requestID)

	// Log the request details
	log.Printf("Request ID: %s | Method: %s | URL: %s",
		requestID,
		c.Method(),
		c.OriginalURL(),
	)

	return c.Next()
}
