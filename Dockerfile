# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o meshtastic-relay ./cmd/relay

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 relay && \
    adduser -u 1000 -G relay -s /bin/sh -D relay

# Create directories
RUN mkdir -p /etc/meshtastic-relay /var/log/meshtastic && \
    chown -R relay:relay /etc/meshtastic-relay /var/log/meshtastic

# Copy binary from builder
COPY --from=builder /app/meshtastic-relay /usr/local/bin/meshtastic-relay

# Copy example config
COPY configs/example.yaml /etc/meshtastic-relay/config.yaml.example

# Switch to non-root user
USER relay

# Set working directory
WORKDIR /home/relay

# Expose default metrics port (if implemented)
# EXPOSE 9090

# Default command
ENTRYPOINT ["meshtastic-relay"]
CMD ["--config", "/etc/meshtastic-relay/config.yaml"]
