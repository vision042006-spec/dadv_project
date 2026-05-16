# ── Build stage for Go API ──────────────────────────────────────
FROM golang:alpine AS go-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/api ./cmd/api/main.go

# ── Runtime stage ─────────────────────────────────────────────
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    redis-server \
    supervisor \
    sqlite3 \
    gcc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy Go API from builder
COPY --from=go-builder /app/api /app/api

# Setup Python worker
COPY cmd/worker/requirements.txt /app/worker/
RUN pip install --no-cache-dir -r /app/worker/requirements.txt
COPY cmd/worker/worker.py /app/worker/

# Setup data directories
RUN mkdir -p /app/data/uploads

# Copy Supervisor config
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Environment variables
ENV REDIS_ADDR=127.0.0.1:6379
ENV REDIS_PASSWORD=""
ENV DATABASE_DSN=/app/data/dadv.db
ENV DATABASE_PATH=/app/data/dadv.db
ENV QUEUE_NAME=metadata_jobs
ENV UPLOAD_DIR=/app/data/uploads
ENV GIN_MODE=release

# Listen on port 10000 for Render
ENV API_PORT=10000
EXPOSE 10000

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
