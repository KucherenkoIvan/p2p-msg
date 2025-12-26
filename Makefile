.PHONY: all \
	fmt \
	clean \
	test \
	deps \
	build \
	protobuf \
	signaling-server \
	tui-client \
	run-signaling-server \
	run-tui-client

# Build all binaries
all: signaling-server tui-client

# Individual service builds
signaling-server:
	@echo "Building signaling server..."
	@go build -o bin/signaling-server cmd/signaling-server/main.go
	@echo "✅ Gateway built: bin/signaling-server"

tui-client:
	@echo "Building TUI client..."
	@go build -o bin/tui-client cmd/tui-client/main.go
	@echo "✅ Client built: bin/tui-client"

# Build all services (excluding client which needs special permissions)
services: signaling-server tui-client

# Build everything
build: all

# Clean binaries
clean:
	@echo "Cleaning binaries..."
	@rm -rf bin/
	@echo "✅ Cleaned"

# Run targets (for development)
run-signaling-server: signaling-server
	@echo "Starting gateway server..."
	@./bin/signaling-server

run-tui-client: tui-client
	@echo "Starting tui client..."
	@./bin/tui-client

# Test targets
test:
	@echo "Running tests..."
	@go test ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...


# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

protobuf:
	@echo "Compiling protobuf..."
	@protoc --proto_path=./pkg/protobuf --go_out=./pkg/protobuf/gen ./pkg/protobuf/*.proto
	@echo "✅ Dependencies installed"

# Help
help:
	@echo "Available targets:"
	@echo "  make all                       - Build all binaries (signaling-server, tui-client)"
	@echo "  make clean                     - Remove all built binaries"
	@echo "  make test                      - Run tests"
	@echo "  make fmt                       - Format code"
	@echo "  make deps                      - Install/update dependencies"
	@echo "  make build                     - Build all binaries (signaling-server, tui-client)"
	@echo "  make protobuf                  - Compile all .proto files"
	@echo "  make signaling-server          - Build signaling server only"
	@echo "  make tui-client                - Build tui-client only"
	@echo ""
	@echo "Run targets (development):"
	@echo "  make run-signaling-server      - Build and run signaling-server"
	@echo "  make run-tui-client            - Build and run tui-client"


