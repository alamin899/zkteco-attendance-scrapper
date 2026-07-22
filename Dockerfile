# -----------------------------------------------------------------------------
# Stage 1: Build stage
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy dependency files for optimal Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build static binary (CGO_ENABLED=0 for maximum portability and zero glibc dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o attendance-collector main.go

# -----------------------------------------------------------------------------
# Stage 2: Final Production Runtime Image
# -----------------------------------------------------------------------------
FROM alpine:3.20

# Install CA certificates (for TLS) and tzdata (for correct timezone conversion)
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root system user and group for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/attendance-collector /app/attendance-collector

# Set ownership
RUN chown -R appuser:appgroup /app

# Run container as non-root user
USER appuser

# Set entrypoint to executable
ENTRYPOINT ["/app/attendance-collector"]

# Default argument is `run` daemon
CMD ["run"]
