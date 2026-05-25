# syntax=docker/dockerfile:1

# Stage 1: Build the SvelteKit app
FROM oven/bun:alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY package.json bun.lock ./

# Install dependencies (frozen-lockfile ensures reproducible builds)
RUN bun install --frozen-lockfile

# Copy the rest of the application files
COPY . .

# Run the SvelteKit build
RUN bun run build

# Stage 2: Production runtime
FROM oven/bun:alpine AS runner

WORKDIR /app

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1001

# Copy built application from builder
COPY --from=builder --chown=appuser:appgroup /app/build ./build

# Set runtime environment variables
ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

# Expose port
EXPOSE 3000

# Use the non-root user
USER 1001:1001

# Start the application
ENTRYPOINT ["bun", "run", "build/index.js"]
