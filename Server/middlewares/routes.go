package middleware

import (
	image "MAIN_SERVER/components/Image"
	"MAIN_SERVER/components/ping"

	"github.com/gofiber/fiber/v2"
)

func LoadRoutes(app *fiber.App) error {

	image.Routes(app)
	ping.Routes(app)
	return nil
}
