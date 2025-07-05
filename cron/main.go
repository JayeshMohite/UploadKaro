package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

var urls = []string{
	"https://test.com/",
}

func main() {
	// Start individual health check routines for each URL
	for _, url := range urls {
		go continuousHealthCheck(url)
	}

	// Start the HTTP server
	if err := http.ListenAndServe("0.0.0.0:8000", nil); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func continuousHealthCheck(url string) {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	log.Printf("Starting continuous health check for %s", url)

	for {
		// Create a new goroutine for each individual request
		go func(checkUrl string) {
			resp, err := client.Get(checkUrl)
			if err != nil {
				log.Printf("Error checking %s: %v", checkUrl, err)
				return
			}
			defer resp.Body.Close()

			_, err = io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response from %s: %v", checkUrl, err)
				return
			}

			log.Printf("Status for %s: %d", checkUrl, resp.StatusCode)
		}(url)

		// Wait for 30 seconds before next check
		time.Sleep(30 * time.Second)
	}
}
