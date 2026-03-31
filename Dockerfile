# --- Stage 1: Build Backend ---
FROM golang:1.24-bookworm AS backend-builder
WORKDIR /app
ENV GOTOOLCHAIN=local
COPY backend/ .
RUN go build -mod=vendor -o main .

# --- Stage 2: Build Frontend ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
# Relative path /api allows the UI to talk to the backend through the Nginx proxy
ARG VITE_API_BASE
ENV VITE_API_BASE=${VITE_API_BASE:-/api} 
RUN npm run build

# --- Stage 3: Final Production Image ---
FROM debian:bullseye-slim
WORKDIR /app

# Install Nginx and Playwright dependencies (Chromium)
RUN apt-get update && apt-get install -y \
    nginx \
    wget \
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
    libcairo-gobject2 \
    libcairo2 \
    libgdk-pixbuf2.0-0 \
    libglib2.0-0 \
    libgtk-3-0 \
    && rm -rf /var/lib/apt/lists/*

# Copy Backend Binary
COPY --from=backend-builder /app/main .
RUN mkdir -p /app/data

# Copy Frontend to Nginx
COPY --from=frontend-builder /app/build /usr/share/nginx/html

# Update Nginx config to handle SPA routing and proxy /api to the backend
RUN cat <<EOF > /etc/nginx/sites-available/default
server {
    listen 4205;
    location / {
        root /usr/share/nginx/html;
        try_files \$uri \$uri/ /index.html;
    }
    location /api {
        proxy_pass http://localhost:4206;
    }
}
EOF

# Create an entrypoint script to run both processes
RUN cat <<EOF > /app/entrypoint.sh
#!/bin/sh
nginx -g 'daemon off;' &
./main
EOF
RUN chmod +x /app/entrypoint.sh

# Environment Defaults
ENV PORT=4206
ENV DATA_DIR=/app/data

# Expose ports
EXPOSE 4205 4206

# Launch the shim
CMD ["/app/entrypoint.sh"]
