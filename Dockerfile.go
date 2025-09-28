# Multi-stage build for minimal image size
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and static files
COPY main.go .
COPY static/ static/

# Build static binary with cache mounts for faster rebuilds
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o epub2kobo

# Download kepubify (optional)
RUN wget -O kepubify.tar.gz https://github.com/pgaskin/kepubify/releases/download/v4.0.4/kepubify-linux-64bit.tar.gz && \
    tar -xzf kepubify.tar.gz kepubify && \
    chmod +x kepubify || true

# Final minimal image
FROM scratch

WORKDIR /

# Copy binary and kepubify
COPY --from=builder /app/epub2kobo /epub2kobo
COPY --from=builder /app/kepubify* /usr/local/bin/

# Create uploads directory
VOLUME ["/uploads"]

EXPOSE 3001

ENTRYPOINT ["/epub2kobo"]
