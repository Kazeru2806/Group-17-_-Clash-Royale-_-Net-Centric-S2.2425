package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// TCPClient represents a TCP client for the game
type TCPClient struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewTCPClient creates a new TCP client
func NewTCPClient(address string) (*TCPClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &TCPClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// Start starts the client
func (c *TCPClient) Start() {
	// Read input from the server in a separate goroutine
	go c.readFromServer()

	// Read input from the user
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.ToUpper(line) == "QUIT" || strings.ToUpper(line) == "EXIT" {
			fmt.Println("Exiting...")
			break
		}

		// Send the line to the server
		if _, err := c.conn.Write([]byte(line + "\n")); err != nil {
			fmt.Printf("Error sending to server: %v\n", err)
			break
		}
	}

	c.conn.Close()
}

// readFromServer reads messages from the server
func (c *TCPClient) readFromServer() {
	for {
		message, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Connection closed: %v\n", err)
			break
		}

		// Print the message
		fmt.Print(message)
	}
}
