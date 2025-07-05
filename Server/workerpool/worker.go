package worker

import (
	"MAIN_SERVER/gcs/queries"
	"fmt"
	"mime/multipart"
	"os"
	"sync"
)

// Declare global channels and WaitGroup for worker synchronization
var (
	TaskChan    chan *Task
	ErrChan     chan error
	WorkerWg    sync.WaitGroup
	WorkerCount = 25 // Set the number of workers (can be adjusted)
)

// Task structure that holds the image and userName
type Task struct {
	File     *multipart.FileHeader
	UserName string
}

// InitializeWorkerPool initializes the worker pool with a fixed number of workers
func InitializeWorkerPool() {
	// Create channels with buffer size
	TaskChan = make(chan *Task, 100) // The buffer size can be adjusted
	ErrChan = make(chan error, 100)

	// Start the worker goroutines
	for i := 0; i < WorkerCount; i++ {
		go Worker(i)
	}
}

// Worker processes tasks (images) from the task channel
func Worker(workerID int) {
	for task := range TaskChan {
		// Process the task (upload image)
		WorkerWg.Add(1)
		fmt.Printf("Worker %d is uploading image: %v for user: %s\n", workerID, task.File.Filename, task.UserName)
		// Simulate the image upload (call your actual upload function here)
		queries.UploadImageToDrive(task.File, os.Getenv("FOLDER_ID"), &WorkerWg, ErrChan, task.UserName)
	}
}
