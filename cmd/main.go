package main

import (
	"context"
	"forum/cmd/config"
	"forum/internal/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Log application start
	log.Println("Starting the forum application...")

	// Create the config object
	configObj := config.CreateConfig()
	if err := config.ReadConfig("cmd/config/Config.json", configObj); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Display config values for debugging purposes
	log.Printf("Config loaded: Address=%s, DB Path=%s, DB Driver=%s", configObj.Address, configObj.DbPath, configObj.DbDriver)

	// Create a context with cancel for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture OS signals for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown signal
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %s. Shutting down server...", sig)
		cancel()
	}()

	// Initialize the server and get the DB connection
	log.Printf("Initializing server on address: %s", configObj.Address)
	srv, db := server.InitServer(configObj, ctx)
	if srv == nil {
		log.Fatal("Server initialization failed (returned nil). Check InitServer function.")
	}

	// Start the server in a goroutine to allow for graceful shutdown
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Set a timeout for graceful shutdown
	shutdownTimeout := 10 * time.Second
	// Wait for the shutdown signal
	<-ctx.Done()

	// Log shutdown
	log.Println("Shutting down the server...")

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Gracefully shut down the server and close the DB
	if err := srv.Shutdown(shutdownCtx, db); err != nil {
		log.Fatalf("Server Shutdown Failed: %v", err)
	}

	log.Println("Server shut down gracefully.")
}
