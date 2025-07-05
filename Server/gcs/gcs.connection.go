package gcs

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var DriveService *drive.Service

// Load Google Drive API client from credentials.json
func GetDriveService() {
	ctx := context.Background()

	// Load credentials file
	b, err := os.ReadFile("creds.json")
	if err != nil {
		fmt.Printf("unable to read client secret file: %v", err)
	}

	// Use the credentials to create a Google Drive client
	config, err := google.JWTConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		fmt.Printf("unable to parse client secret file to config: %v", err)
	}

	client := config.Client(ctx)
	DriveService, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Printf("unable to retrieve Drive client: %v", err)
	}

	fmt.Println("Connected to Google Drive API")
}
