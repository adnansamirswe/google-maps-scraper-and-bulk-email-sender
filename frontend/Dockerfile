# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY package.json package-lock.json ./
RUN npm install

# Copy source code
COPY . .

# Set dynamic API base for production build
# This forces Vite to baked in the correct URL if it's not relative
ARG VITE_API_BASE
ENV VITE_API_BASE=${VITE_API_BASE:-http://localhost:4206}

# Build the project
RUN npm run build

# Runtime stage
FROM nginx:alpine

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy build output to Nginx's serve directory
COPY --from=builder /app/build /usr/share/nginx/html

# Expose the frontend port
EXPOSE 4205

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
