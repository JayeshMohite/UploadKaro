package image

import "github.com/gofiber/fiber/v2"

func Routes(app *fiber.App) {
	grp := app.Group("/image")

	grp.Post("/upload", uploadImageController)
	grp.Post("/listing", listingImageController)
	grp.Post("/like", likeImageController)

}
