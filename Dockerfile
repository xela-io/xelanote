# Multi-stage build: Frontend + Backend

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy built frontend from stage 1
COPY --from=frontend-builder /frontend/build ./cmd/server/static/

# Verify static files were copied (debugging)
RUN ls -la ./cmd/server/static/ && echo "Static files found: $(find ./cmd/server/static/ -type f | wc -l) files"

# Build with CGO and FTS5 for SQLite (stripped for production)
ARG VERSION=dev
RUN CGO_ENABLED=1 go build -tags "fts5" \
  -ldflags="-s -w -X github.com/xela-io/xelanote/internal/api.Version=${VERSION}" \
  -o /xelanote ./cmd/server

# Verify binary size
RUN ls -lh /xelanote && echo "Binary size: $(du -h /xelanote | cut -f1)"

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=backend-builder /xelanote /app/xelanote
COPY CHANGELOG.md /app/CHANGELOG.md

# Create data directory and set ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Environment
ENV XELANOTE_DB=/app/data/xelanote.db

EXPOSE 8080

VOLUME ["/app/data"]

# Run as non-root user
USER appuser

# Health check for container orchestration
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["/app/xelanote", "-addr", ":8080"]
