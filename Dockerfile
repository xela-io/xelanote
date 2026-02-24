# Multi-stage build: Frontend + Backend

# Stage 1: Build Frontend
# F2-05: Digest-pinned base images. Update quarterly: docker pull <image>, then
# update the digest with: docker inspect --format='{{index .RepoDigests 0}}' <image>
# Last updated: 2026-02-24
FROM node:22-alpine@sha256:e4bf2a82ad0a4037d28035ae71529873c069b13eb0455466ae0bc13363826e34 AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
# Last updated: 2026-02-24
FROM golang:1.25-alpine@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS backend-builder

# For SQLCipher support: replace "gcc musl-dev" with "gcc musl-dev sqlcipher-dev",
# change build tags to "fts5 sqlite_crypt", and add "sqlcipher" to runtime stage.
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy built frontend from stage 1
COPY --from=frontend-builder /frontend/build ./cmd/server/static/

# Build with CGO and FTS5 for SQLite (stripped for production)
ARG VERSION=dev
RUN CGO_ENABLED=1 go build -tags "fts5" \
  -ldflags="-s -w -X github.com/xela-io/xelanote/internal/api.Version=${VERSION}" \
  -o /xelanote ./cmd/server

# Stage 3: Runtime
# Last updated: 2026-02-24
FROM alpine:3.20@sha256:a4f4213abb84c497377b8544c81b3564f313746700372ec4fe84653e4fb03805

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
