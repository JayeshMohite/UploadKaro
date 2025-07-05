package image

import (
	"MAIN_SERVER/components/Image/dto"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func uploadImageController(c *fiber.Ctx) error {

	var imageDto dto.ImageReqDto

	// Parse the form fields and files
	if err := c.BodyParser(&imageDto); err != nil {
		fmt.Println("Error parsing request body:", err)
		return err
	}

	// Retrieve the uploaded images using FormFile method
	// Fiber allows you to get files via `c.FormFile(fieldName)`
	files, err := c.MultipartForm()
	if err != nil {
		fmt.Println("Error retrieving multipart form:", err)
		return err
	}

	// Assuming images are under the field name "images"
	imageFiles := files.File["images"]
	if len(imageFiles) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "No images uploaded")
	}

	// Send the image upload task to a background worker
	go func() {
		uploadImages(imageFiles, imageDto.UserName)
	}()

	// Immediately respond to the client that the images are being uploaded
	return c.SendString("Images upload started successfully, processing in the background.")
}

func listingImageController(c *fiber.Ctx) error {

	var ImageListingReqDto dto.ImageListingReqDto

	// Parse the form fields and files
	if err := c.BodyParser(&ImageListingReqDto); err != nil {
		fmt.Println("Error parsing request body:", err)
		return err
	}

	// Retrieve the uploaded images using FormFile method
	files, totalPages, err := ListImages(ImageListingReqDto.PageNumber, ImageListingReqDto.PageSize, ImageListingReqDto.OrderBy)
	if err != nil {
		fmt.Println("Error retrieving multipart form:", err)
		return err
	}

	// Return the files as a JSON response, including the nextPageToken for pagination
	return c.JSON(fiber.Map{
		"files":       files,
		"total_pages": totalPages,
	})
}

func likeImageController(c *fiber.Ctx) error {

	var imageLikeDto dto.ImageLikeReqDto

	// Parse the form fields and files
	if err := c.BodyParser(&imageLikeDto); err != nil {
		fmt.Println("Error parsing request body:", err)
		return err
	}

	likeCount, err := likeImage(imageLikeDto.ImageID)
	if err != nil {
		fmt.Println("Error Liking image :", err)
		return err
	}

	return c.JSON(fiber.Map{
		"liked_count": likeCount,
	})
}
