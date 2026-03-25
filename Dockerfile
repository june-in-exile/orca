# Stage 1: Build
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile binary
RUN go build -o paylock ./cmd/paylock

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install ffmpeg and required packages
RUN apt-get update && apt-get install -y \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/paylock .

# Set default environment variables
ENV PAYLOCK_PORT=8080
ENV PAYLOCK_DATA_DIR=/data

# Create data directory
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Start command
CMD ["./paylock"]
