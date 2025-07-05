package main

import (
	"log"
	"strings"
	"sync/atomic"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/google/uuid"
)

type LoadBalancer struct {
	Backends []string
	counter  uint32
}

// NewLoadBalancer initializes a load balancer with backend URLs
func NewLoadBalancer(backends []string) *LoadBalancer {
	return &LoadBalancer{Backends: backends}
}

// GetBackend selects the next backend based on round-robin
func (lb *LoadBalancer) GetBackend() string {
	index := atomic.AddUint32(&lb.counter, 1)
	return lb.Backends[index%uint32(len(lb.Backends))]
}

func main() {

	// middleware.LoadConfig("config.json")

	backendStr := "https://test.com/"

	backends := strings.Split(backendStr, ",")

	// Initialize load balancer
	lb := NewLoadBalancer(backends)

	// Set up Fiber
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET, POST",
	}))

	// Middleware to add a unique request ID and log the request
	app.All("/*", func(c *fiber.Ctx) error {
		// Generate a unique request ID
		requestID := uuid.New().String()

		// Store the request ID in the context to pass it to the backend
		c.Set("X-Request-ID", requestID)

		// Select the backend
		backend := lb.GetBackend()

		// Log the request with ID and target backend
		log.Printf("Request ID: %s | Forwarding request to: %s", requestID, backend)

		// Use Fiber's proxy middleware to forward the request, including the request ID
		c.Request().Header.Set("X-Request-ID", requestID)

		// Forward the full URL, including path, to the backend
		targetURL := backend + c.OriginalURL()
		return proxy.Do(c, targetURL)
	})

	// Start the load balancer server
	if err := app.Listen("0.0.0.0:3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
