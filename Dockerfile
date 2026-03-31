# --- Stage 1: Build Backend ---
FROM golang:1.22-bullseye AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN go build -o main .

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
RUN echo 'server {\n\
    listen 4205;\n\
    location / {\n\
        root /usr/share/nginx/html;\n\
        try_files $uri $uri/ /index.html;\n\
    }\n\
    location /api {\n\
        proxy_pass http://localhost:4206;\n\
    }\n\
}' > /etc/nginx/sites-available/default

# Create an entrypoint script to run both processes
RUN echo "#!/bin/sh\nnginx -g 'daemon off;' & \n./main" > /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Environment Defaults
ENV PORT=4206
ENV DATA_DIR=/app/data

# Expose ports
EXPOSE 4205 4206

# Launch the shim
CMD ["/app/entrypoint.sh"]
