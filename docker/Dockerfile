# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

RUN apk add --no-cache make && \
	apk add --no-cache protobuf-dev && \
	apk add --no-cache protoc

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all required binaries
RUN make all

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/signaling-server /app/bin/signaling

# Copy configs
COPY cfg /app/cfg

# Copy entrypoint script
# COPY emulation/docker/gateway-emulation/entrypoint.sh /app/entrypoint.sh
# RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/bin/signaling"]


