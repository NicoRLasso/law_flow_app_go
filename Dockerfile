# CSS build stage
FROM node:20-alpine AS css-builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

# Copy files needed for Tailwind to scan classes
COPY static/css/input.css static/css/kinetic.css ./static/css/
COPY templates ./templates
COPY postcss.config.js ./
RUN npm run build:css

# Go build stage - use Debian for glibc compatibility with runtime
FROM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*

# Install templ CLI
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy built CSS from css-builder stage
COPY --from=css-builder /app/static/css/bundle.min.css ./static/css/bundle.min.css

# Generate templ files
RUN templ generate

# Build with CGO for SQLite (uses cached dependencies)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o server cmd/server/main.go

# Headless shell stage - get Chrome headless binary
FROM chromedp/headless-shell:latest AS headless

# Runtime stage - Debian slim for glibc compatibility
FROM debian:bookworm-slim

# Install runtime dependencies for headless-shell
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    fonts-dejavu-core \
    libnss3 \
    libnspr4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy headless-shell from chromedp image
COPY --from=headless /headless-shell /headless-shell

# Set Chrome path for chromedp
ENV CHROME_PATH=/headless-shell/headless-shell

WORKDIR /app

# Copy binary and required files
COPY --from=builder /app/server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/services/i18n ./services/i18n

# Create directories for data and a non-root user
RUN mkdir -p /app/db /app/uploads && \
    useradd --no-create-home --home-dir /app appuser && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["./server"]

