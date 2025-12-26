# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all required binaries
RUN go build -o bin/signaling cmd/signaling-server/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/signaling /app/bin/signaling

# Copy configs
COPY cfg /app/cfg

# Copy entrypoint script
# COPY emulation/docker/gateway-emulation/entrypoint.sh /app/entrypoint.sh
# RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/bin/signaling"]


