package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/user/tcr/server"
)

func main() {
	// Parse command-line flags
	address := flag.String("address", ":8080", "Server address to listen on")
	flag.Parse()

	// Create data directories if they don't exist
	if err := os.MkdirAll("data/players", 0755); err != nil {
		log.Fatalf("Error creating data directories: %v", err)
	}

	// Create server
	srv, err := server.NewTCPServer(*address)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	// Start server
	log.Printf("Starting server on %s", *address)
	srv.Start()
}
