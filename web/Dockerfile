# Build stage
FROM node:22-alpine AS build

WORKDIR /app

# Copy package files
COPY ./web/package.json ./web/package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source code
COPY ./web/ .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy custom nginx config
COPY ./web/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from build stage
COPY --from=build /app/dist /usr/share/nginx/html

# Copy env script for runtime config injection
COPY ./web/env.sh /docker-entrypoint.d/env.sh
RUN sed -i 's/\r$//' /docker-entrypoint.d/env.sh && chmod +x /docker-entrypoint.d/env.sh

# Expose port 80
EXPOSE 80

# Start nginx (env.sh runs automatically via docker-entrypoint.d)
CMD ["nginx", "-g", "daemon off;"]
