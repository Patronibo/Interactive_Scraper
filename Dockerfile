# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files from backend directory
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copy source code
COPY backend/ ./

# Update go.mod and build the application
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /app/main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy frontend files with explicit layer separation for better cache invalidation
# Frontend changes will invalidate cache and trigger rebuild
COPY frontend/index.html ./frontend/index.html
COPY frontend/static/ ./frontend/static/

# Add build metadata (helps with cache invalidation)
ARG BUILD_DATE
ARG BUILD_VERSION
LABEL build.date="${BUILD_DATE}" \
      build.version="${BUILD_VERSION}" \
      maintainer="CTI Dashboard"

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
