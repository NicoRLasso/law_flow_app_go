# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with CGO for SQLite
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o server cmd/server/main.go

# Runtime stage - minimal image
FROM alpine:3.21

# Install runtime dependencies including Chromium for PDF generation
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    font-noto-emoji

# Set Chromium flags for headless operation
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROME_PATH=/usr/lib/chromium/

WORKDIR /app

# Copy binary and required files
COPY --from=builder /app/server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/services/i18n ./services/i18n

# Create directories for data and a non-root user
RUN mkdir -p /app/db /app/uploads && \
    adduser -D -h /app appuser && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["./server"]

