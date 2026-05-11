package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"socks5-dpi-proxy/internal/api/handlers"
	"socks5-dpi-proxy/internal/api/middleware"
	"socks5-dpi-proxy/internal/pipeline"
	"socks5-dpi-proxy/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var (
		port = flag.String("port", "8080", "API server port")
	)
	flag.Parse()

	// Initialize storage
	storage, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize ML engine
	mlEngine := pipeline.NewMLPipelineEngine()

	// Initialize handlers
	mlHandler := handlers.NewMLHandler(mlEngine, storage)
	requestHandler := handlers.NewRequestHandler(storage)
	wsHandler := handlers.NewWebSocketHandler()

	// Setup Gin router
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// ML endpoints
		ml := v1.Group("/ml")
		{
			ml.GET("/status", mlHandler.GetStatus)
			ml.GET("/statistics", mlHandler.GetStatistics)
			ml.GET("/techniques", mlHandler.GetTechniques)
			ml.GET("/history/:hours", mlHandler.GetHistory)

			// Control endpoints
			ml.POST("/feedback", mlHandler.PostFeedback)
			ml.PUT("/techniques/:technique/effectiveness", mlHandler.UpdateTechniqueEffectiveness)
			ml.POST("/retrain", mlHandler.RetrainModel)
			ml.PUT("/config", mlHandler.UpdateConfig)
		}

		// Request tracing endpoints
		requests := v1.Group("/requests")
		{
			requests.GET("", requestHandler.GetRequests)
			requests.GET("/:id", requestHandler.GetRequestDetails)
		}

		// Domain statistics
		domains := v1.Group("/domains")
		{
			domains.GET("/:domain/statistics", requestHandler.GetDomainStatistics)
		}

		// WebSocket endpoint
		v1.GET("/ws", wsHandler.HandleWebSocket)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
