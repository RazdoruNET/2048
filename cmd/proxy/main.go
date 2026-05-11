package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"socks5-dpi-proxy/internal/proxy"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":1080", "SOCKS5 proxy listen address")
		configPath = flag.String("config", "configs/proxy.yaml", "Configuration file path")
	)
	flag.Parse()

	server := proxy.NewServer(*listenAddr, *configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	log.Printf("Starting SOCKS5 DPI proxy on %s", *listenAddr)
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
