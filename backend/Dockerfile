# Multi-stage build: Frontend + Backend

# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend (SQLCipher enabled)
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev sqlcipher-dev

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy built frontend from stage 1
COPY --from=frontend-builder /frontend/build ./cmd/server/static/

# Build with CGO, SQLCipher, and FTS5 for SQLite
ARG VERSION=dev
RUN CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" \
  -ldflags="-s -w -X github.com/xela-io/xelanote/internal/api.Version=${VERSION}" \
  -o /xelanote ./cmd/server

# Stage 3: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata sqlcipher

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=backend-builder /xelanote /app/xelanote

# Create data directory and set ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Environment
ENV XELANOTE_DB=/app/data/xelanote.db

EXPOSE 8080

VOLUME ["/app/data"]

# Run as non-root user
USER appuser

CMD ["/app/xelanote", "-addr", ":8080"]
