package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/user/tcr/client"
)

func main() {
	// Parse command-line flags
	address := flag.String("address", "localhost:8080", "Server address to connect to")
	flag.Parse()

	// Create client
	fmt.Printf("Connecting to server at %s\n", *address)
	client, err := client.NewTCPClient(*address)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}

	fmt.Println("Connected. Type commands to interact with the server. Type 'HELP' for a list of commands.")
	fmt.Println("Type 'QUIT' or 'EXIT' to exit.")

	// Start client
	client.Start()
}
