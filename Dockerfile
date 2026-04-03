# --- Stage 1: Build Backend ---
FROM golang:1.24-bookworm AS backend-builder
WORKDIR /app
COPY backend/ .
RUN go build -mod=vendor -o main .

# --- Stage 2: Build Frontend ---
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json ./
RUN npm install
COPY frontend/ .
ARG VITE_API_BASE=""
ENV VITE_API_BASE=${VITE_API_BASE}
RUN npm run build

# --- Stage 3: Final Production Image ---
FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    nginx \
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

COPY --from=backend-builder /app/main .
RUN mkdir -p /app/data

COPY --from=frontend-builder /app/build /usr/share/nginx/html

RUN cat <<'NGINX' > /etc/nginx/sites-available/default
server {
    listen 3000;

    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:4206;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        chunked_transfer_encoding on;
    }
}
NGINX

RUN cat <<'ENTRY' > /app/entrypoint.sh
#!/bin/sh
./main &
exec nginx -g 'daemon off;'
ENTRY
RUN chmod +x /app/entrypoint.sh

ENV PORT=4206
ENV DATA_DIR=/app/data

EXPOSE 3000

CMD ["/app/entrypoint.sh"]
