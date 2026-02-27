# Build stage
FROM node:24-alpine as build-stage

WORKDIR /app

# Set Node memory limit and Docker build flag
ENV NODE_OPTIONS="--max-old-space-size=4096"
ENV VITE_IS_DOCKER=true

# Copy dependency files
COPY package*.json ./
COPY packages/xlsx-0.20.2.tgz ./packages/xlsx-0.20.2.tgz

# Install dependencies
RUN corepack enable
RUN pnpm install

# Copy project files
COPY . .

# Build application
RUN pnpm run build

# Production stage
FROM nginx:stable-alpine as production-stage

# Copy build output to nginx html directory
COPY --from=build-stage /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
