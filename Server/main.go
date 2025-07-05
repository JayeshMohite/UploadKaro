package main

import (
	"MAIN_SERVER/gcs"
	middleware "MAIN_SERVER/middlewares"
	postgresql "MAIN_SERVER/postgress"
	postgressqueries "MAIN_SERVER/postgress/queries"
	worker "MAIN_SERVER/workerpool"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	// Set up Fiber
	app := fiber.New()

	// Register middleware
	app.Use(middleware.RequestIDLogger)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET, POST",
	}))

	middleware.LoadConfig("config.json")
	middleware.LoadRoutes(app)

	postgresql.PostgresDbConnect()
	worker.InitializeWorkerPool()

	// start google drive connection
	gcs.GetDriveService()

	// signal channel to capture system calls
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// start shutdown goroutine
	go func() {
		// capture sigterm and other system call here
		<-sigCh
		postgressqueries.GetLikeCache().Stop()

		fmt.Println("Shutting down...")
		_ = app.Shutdown()
	}()

	// start http server
	if err := app.Listen("0.0.0.0:3001"); err != nil {
		fmt.Println("Error starting server:", err)
		panic(err)
	}
}
