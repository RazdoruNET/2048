# Build stage
FROM golang:1.25-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/proxy

# Final stage
FROM alpine:3.19

# Install ca-certificates and security updates
RUN apk --no-cache add ca-certificates tzdata && \
    apk update && \
    apk upgrade

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership
RUN chown -R appuser:appgroup /root/

USER appuser

# Expose SOCKS5 port
EXPOSE 1080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 1080 || exit 1

# Run the binary
CMD ["./main"]
