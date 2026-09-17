.PHONY: build build-server build-client run-server run-client clean

# Build both server and client
build: build-server build-client

# Build server
build-server:
	@echo "Building server..."
	go build -o bin/tcr-server ./cmd/server

# Build client
build-client:
	@echo "Building client..."
	go build -o bin/tcr-client ./cmd/client

# Run server
run-server:
	@echo "Running server..."
	./bin/tcr-server

# Run client
run-client:
	@echo "Running client..."
	./bin/tcr-client

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Create necessary directories
init:
	@echo "Creating necessary directories..."
	mkdir -p data/players
	mkdir -p bin 