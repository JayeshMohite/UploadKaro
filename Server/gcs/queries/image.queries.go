package queries

import (
	"MAIN_SERVER/gcs"
	postgressqueries "MAIN_SERVER/postgress/queries"
	"fmt"
	"log"
	"mime/multipart"
	"sync"
	"time"

	"google.golang.org/api/drive/v3"
)

func UploadImageToDrive(file *multipart.FileHeader, folderID string, wg *sync.WaitGroup, errChan chan<- error, userName string) {

	defer wg.Done()

	// Open the file
	fileContent, err := file.Open()
	if err != nil {
		errChan <- fmt.Errorf("unable to open file %s: %v", file.Filename, err)
		return
	}
	defer fileContent.Close()

	// Set file metadata with the new unique file name
	fileMetadata := &drive.File{
		Name: file.Filename,
	}

	if folderID != "" {
		fileMetadata.Parents = []string{folderID}
	}

	// Upload the file
	uploadedFile, err := gcs.DriveService.Files.Create(fileMetadata).
		Media(fileContent). // Upload using the file content as Media
		Do()
	if err != nil {
		errChan <- fmt.Errorf("unable to upload file %s: %v", file.Filename, err)
		return
	}

	// Generate the download URL (can be fetched by users for downloading)
	downloadURL := fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", uploadedFile.Id)

	// Wait for the thumbnail to be generated
	for i := 0; i < 5; i++ {
		// Fetch the file metadata again to get the thumbnail link
		fileMetadata, err = gcs.DriveService.Files.Get(uploadedFile.Id).Fields("thumbnailLink").Do()
		if err != nil {
			errChan <- fmt.Errorf("unable to fetch file metadata for %s: %v", file.Filename, err)
			return
		}

		if fileMetadata.ThumbnailLink != "" {
			break
		}

		// Wait for 1 second before trying again
		time.Sleep(1 * time.Second)
	}

	err = postgressqueries.InsertImageID(uploadedFile.Id, userName, uploadedFile.Name, downloadURL, fileMetadata.ThumbnailLink)
	if err != nil {
		errChan <- err
		return
	}

	log.Printf("Uploaded file %s with ID: %s", uploadedFile.Name, uploadedFile.Id)
}
