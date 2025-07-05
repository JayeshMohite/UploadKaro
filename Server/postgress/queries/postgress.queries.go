package postgressqueries

import (
	"MAIN_SERVER/components/Image/dto"
	"MAIN_SERVER/gcs"
	postgresql "MAIN_SERVER/postgress"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

func LikeImageCount(imageID string) (int, error) {

	postgresql.PostgresDbConnect()
	var likeCount int

	err := postgresql.PostgresConnection.QueryRow("SELECT liked_count FROM images WHERE image_id = $1", imageID).Scan(&likeCount)
	if err != nil {
		return 0, err
	}

	return likeCount, nil
}

type CacheEntry struct {
	baseCount    int
	cacheCount   int
	lastAccessed time.Time
}

type LikeCache struct {
	cache          map[string]*CacheEntry
	pendingUpdates map[string]bool
	mutex          sync.RWMutex
	done           chan bool

	// Configuration
	maxEntries      int
	cleanupInterval time.Duration
	maxIdleTime     time.Duration
}

var (
	likeCache *LikeCache
	once      sync.Once
)

// GetLikeCache returns singleton instance
func GetLikeCache() *LikeCache {
	once.Do(func() {
		likeCache = &LikeCache{
			cache:           make(map[string]*CacheEntry),
			pendingUpdates:  make(map[string]bool),
			done:            make(chan bool),
			maxEntries:      100000,
			cleanupInterval: 2 * time.Minute,
			maxIdleTime:     5 * time.Minute,
		}
		go likeCache.startBackgroundProcessor()
		go likeCache.startCleanupProcessor()
	})
	return likeCache
}

// startBackgroundProcessor runs periodic updates to persist likes to database
func (c *LikeCache) startBackgroundProcessor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.processPendingLikes()
		case <-c.done:
			return
		}
	}
}

// updateDatabaseLikes updates the like count in the database
func (c *LikeCache) updateDatabaseLikes(imageID string, likeCount int) error {
	_, err := postgresql.PostgresConnection.Exec(`
		UPDATE images
		SET liked_count = liked_count + $1
		WHERE image_id = $2
	`, likeCount, imageID)

	return err
}

func (c *LikeCache) getInitialCount(imageID string) (int, error) {
	var count int
	err := postgresql.PostgresConnection.QueryRow(`
		SELECT liked_count 
		FROM images 
		WHERE image_id = $1
	`, imageID).Scan(&count)

	return count, err
}

func (c *LikeCache) LikeImage(imageID string) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.cache[imageID]
	if !exists {
		// Check cache size before adding new entry
		if len(c.cache) >= c.maxEntries {
			c.evictOldestEntries(100) // Remove 100 oldest entries
		}

		dbCount, err := c.getInitialCount(imageID)
		if err != nil {
			return 0, err
		}

		entry = &CacheEntry{
			baseCount:    dbCount,
			cacheCount:   0,
			lastAccessed: time.Now(),
		}
		c.cache[imageID] = entry
	}

	entry.cacheCount++
	entry.lastAccessed = time.Now()
	c.pendingUpdates[imageID] = true

	return entry.baseCount + entry.cacheCount, nil
}

func (c *LikeCache) evictOldestEntries(count int) {
	type entryAge struct {
		id       string
		accessed time.Time
	}

	entries := make([]entryAge, 0, len(c.cache))
	for id, entry := range c.cache {
		if !c.pendingUpdates[id] { // Don't evict entries with pending updates
			entries = append(entries, entryAge{id, entry.lastAccessed})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].accessed.Before(entries[j].accessed)
	})

	for i := 0; i < count && i < len(entries); i++ {
		delete(c.cache, entries[i].id)
	}
}

func (c *LikeCache) processPendingLikes() {
	c.mutex.Lock()
	updates := make(map[string]int)

	for imageID := range c.pendingUpdates {
		if entry, exists := c.cache[imageID]; exists {
			updates[imageID] = entry.cacheCount
			entry.baseCount += entry.cacheCount
			entry.cacheCount = 0
		}
		delete(c.pendingUpdates, imageID)
	}

	c.mutex.Unlock()

	if len(updates) > 0 {
		for imageID, likeCount := range updates {
			err := c.updateDatabaseLikes(imageID, likeCount)
			if err != nil {
				c.mutex.Lock()
				if entry, exists := c.cache[imageID]; exists {
					entry.cacheCount += likeCount
					entry.baseCount -= likeCount
					c.pendingUpdates[imageID] = true
				}
				c.mutex.Unlock()
			}
		}
	}
}

func (c *LikeCache) startCleanupProcessor() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupIdleEntries()
		case <-c.done:
			return
		}
	}
}

func (c *LikeCache) cleanupIdleEntries() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for id, entry := range c.cache {
		if !c.pendingUpdates[id] &&
			now.Sub(entry.lastAccessed) > c.maxIdleTime {
			delete(c.cache, id)
		}
	}
}

func (c *LikeCache) Stop() {
	c.done <- true
	c.processPendingLikes() // Process any remaining updates
}

func InsertImageID(imageID string, userName string, fileName string, downloadURL string, thumbnailLink string) error {
	postgresql.PostgresDbConnect()

	_, err := postgresql.PostgresConnection.Exec(
		"INSERT INTO images (image_id, created_at, liked_count,uploaded_by, file_name, download_url, thumbnail_link, is_approved, marked_for_review) VALUES ($1, CURRENT_TIMESTAMP, 0, $2, $3, $4, $5,0,0)",
		imageID,
		userName,
		fileName,
		downloadURL,
		thumbnailLink,
	)
	if err != nil {
		fmt.Println("Error inserting image ID:", err)
		return err
	}
	return nil
}

// WorkerPool manages a fixed pool of workers for thumbnail validation
type WorkerPool struct {
	workerCount int
	tasks       chan ValidationTask
	results     chan ValidationResult
	client      *http.Client
}

type ValidationTask struct {
	Index     int
	ImageID   string
	Thumbnail string
}

type ValidationResult struct {
	Index   int
	ImageID string
	IsValid bool
	NewLink string
}

var (
	// Global worker pool instance
	globalWorkerPool *WorkerPool
	poolOnce         sync.Once
)

// GetWorkerPool returns the singleton worker pool instance
func GetWorkerPool() *WorkerPool {
	poolOnce.Do(func() {
		globalWorkerPool = NewWorkerPool(5) // Adjust worker count based on your needs
	})
	return globalWorkerPool
}

func NewWorkerPool(workerCount int) *WorkerPool {
	wp := &WorkerPool{
		workerCount: workerCount,
		tasks:       make(chan ValidationTask, workerCount*2),
		results:     make(chan ValidationResult, workerCount*2),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	wp.Start()
	return wp
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		go wp.worker()
	}
}

func (wp *WorkerPool) worker() {
	for task := range wp.tasks {
		result := ValidationResult{
			Index:   task.Index,
			ImageID: task.ImageID,
			IsValid: true,
		}

		// Check if thumbnail is accessible
		resp, err := wp.client.Head(task.Thumbnail)
		if err != nil || resp.StatusCode != http.StatusOK {
			result.IsValid = false
			// Try to refresh the thumbnail
			newLink, refreshErr := RefreshThumbnailLink(task.ImageID)
			if refreshErr == nil {
				result.NewLink = newLink
			}
		}

		wp.results <- result
	}
}

func ListImages(pageNumber int, pageSize int, orderBy string) ([]dto.FileResponse, int, error) {
	validOrderBys := map[string]bool{
		"liked_count": true,
		"created_at":  true,
	}

	if !validOrderBys[orderBy] {
		orderBy = "liked_count"
	}

	offset := pageNumber * pageSize

	query := fmt.Sprintf(`
        SELECT image_id, file_name, thumbnail_link, liked_count, download_url
        FROM images
		WHERE is_approved = 1
        ORDER BY %s DESC
        LIMIT $1 OFFSET $2
    `, orderBy)

	rows, err := postgresql.PostgresConnection.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to query database: %v", err)
	}
	defer rows.Close()

	var fileResponses []dto.FileResponse
	wp := GetWorkerPool()
	taskCount := 0

	// First pass: collect all responses and submit validation tasks
	for i := 0; rows.Next(); i++ {
		var file dto.FileResponse
		if err := rows.Scan(&file.ID, &file.Name, &file.Thumbnail, &file.LikedCount, &file.DownloadURL); err != nil {
			return nil, 0, fmt.Errorf("unable to scan row: %v", err)
		}
		fileResponses = append(fileResponses, file)

		// Submit validation task
		wp.tasks <- ValidationTask{
			Index:     i,
			ImageID:   file.ID,
			Thumbnail: file.Thumbnail,
		}
		taskCount++
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %v", err)
	}

	// Collect validation results
	for i := 0; i < taskCount; i++ {
		result := <-wp.results
		if !result.IsValid && result.NewLink != "" {
			// Update the database in background
			go func(imageID, newLink string) {
				if err := updateThumbnailInDB(imageID, newLink); err != nil {
					log.Printf("Error updating thumbnail for image %s: %v", imageID, err)
				}
			}(result.ImageID, result.NewLink)

			// Update the response immediately
			fileResponses[result.Index].Thumbnail = result.NewLink
		}
	}

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM images`
	err = postgresql.PostgresConnection.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to get total count: %v", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return fileResponses, totalPages, nil
}

// updateThumbnailInDB updates the thumbnail link in the database
func updateThumbnailInDB(imageID, newThumbnail string) error {
	query := `
        UPDATE images 
        SET thumbnail_link = $1 
        WHERE image_id = $2
    `

	_, err := postgresql.PostgresConnection.Exec(query, newThumbnail, imageID)
	return err
}

// RefreshThumbnailLink gets a fresh thumbnail link from Google Drive
func RefreshThumbnailLink(fileId string) (string, error) {
	file, err := gcs.DriveService.Files.Get(fileId).
		Fields("thumbnailLink", "webContentLink").
		Do()
	if err != nil {
		return "", fmt.Errorf("error refreshing thumbnail link: %v", err)
	}

	// First try to get thumbnailLink
	if file.ThumbnailLink != "" {
		return file.ThumbnailLink, nil
	}

	// Fallback to webContentLink if available
	if file.WebContentLink != "" {
		return file.WebContentLink, nil
	}

	return "", fmt.Errorf("no valid thumbnail or web content link available")
}
