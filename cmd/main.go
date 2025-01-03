package main

import (
	"context"
	"forum/cmd/config"
	"forum/internal/server"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %s. Shutting down server...", sig)
		cancel()
	}()

	// Initialize the server
	log.Printf("Initializing server on address: %s", configObj.Address)
	srv := server.InitServer(configObj, ctx)
	if srv == nil {
		log.Fatal("Server initialization failed (returned nil). Check InitServer function.")
	}

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

	log.Println("Server shut down gracefully.")
}
