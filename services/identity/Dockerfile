# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy shared pkg and service code
COPY pkg/ ./pkg/
COPY services/identity/ ./services/identity/

# Build the static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/identity services/identity/main.go

FROM alpine:3.20.3

# Create a non-root group and user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1000

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/identity .

# Use non-root user
USER 1000:1000

HEALTHCHECK NONE

ENTRYPOINT ["./identity"]
