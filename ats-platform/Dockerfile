# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the service (default to resume-service, can be overridden with TARGET_SERVICE)
ARG TARGET_SERVICE=resume-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/${TARGET_SERVICE}

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Set timezone
ENV TZ=Asia/Shanghai

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Copy configs
COPY configs/ /app/configs/

# Expose port (default 8081, can be overridden)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${SERVER_PORT:-8081}/health || exit 1

# Run the service
ENTRYPOINT ["/app/server"]
