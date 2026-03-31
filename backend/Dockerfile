# Build Stage
FROM golang:1.22-bullseye AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o main .

# Runtime Stage
FROM debian:bullseye-slim

WORKDIR /app

# Install dependencies for Playwright (Chromium)
RUN apt-get update && apt-get install -y \
    wget \
    gnupg \
    ca-certificates \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpangocairo-1.0-0 \
    libpango-1.0-0 \
    libatk1.0-0 \
    libcairo-gobject2 \
    libcairo2 \
    libgdk-pixbuf2.0-0 \
    libglib2.0-0 \
    libgtk-3-0 \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/main .

# Create data directory
RUN mkdir -p /app/data

# Set environment variables
ENV PORT=4206
ENV DATA_DIR=/app/data
ENV APP_PASSWORD=admin123
ENV JWT_SECRET=change-me-in-production

# Expose backend port
EXPOSE 4206

# Run the server
CMD ["./main"]
