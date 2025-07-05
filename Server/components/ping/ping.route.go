package ping

import "github.com/gofiber/fiber/v2"

func Routes(app *fiber.App) {

	grp := app.Group("/ping")

	grp.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

}
