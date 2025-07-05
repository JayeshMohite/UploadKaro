package image

import (
	"MAIN_SERVER/components/Image/dto"
	postgressqueries "MAIN_SERVER/postgress/queries"
	worker "MAIN_SERVER/workerpool"
	"log"
	"mime/multipart"
)

func uploadImages(image []*multipart.FileHeader, userName string) error {

	// Add images to the task queue
	for _, file := range image {
		task := &worker.Task{
			File:     file,
			UserName: userName,
		}
		worker.TaskChan <- task
	}

	for err := range worker.ErrChan {
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}

	return nil
}
func ListImages(pageNumber int, pageSize int, orderBy string) ([]dto.FileResponse, int, error) {
	return postgressqueries.ListImages(pageNumber, pageSize, orderBy)
}

func likeImage(imageID string) (int, error) {
	return postgressqueries.GetLikeCache().LikeImage(imageID)
}
