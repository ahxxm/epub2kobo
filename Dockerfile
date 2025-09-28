FROM golang:alpine AS builder

RUN apk add --no-cache upx

WORKDIR /app

# Download kepubify for embedding
RUN wget -O kepubify https://github.com/pgaskin/kepubify/releases/download/v4.0.4/kepubify-linux-64bit && \
    chmod +x kepubify

COPY go.mod go.sum* ./
RUN go mod download

COPY main.go .
COPY static/ static/

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o epub2kobo && \
    upx --best --lzma epub2kobo

FROM scratch

WORKDIR /

COPY --from=builder /app/epub2kobo /epub2kobo

EXPOSE 3001

ENTRYPOINT ["/epub2kobo"]
