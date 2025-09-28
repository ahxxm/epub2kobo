# Send to Kobo

A lightweight web service for sending EPUB files to Kobo e-readers, with optional KEPUB conversion.

## Features

- Upload EPUB files from desktop/mobile to Kobo e-reader
- Optional KEPUB conversion for enhanced reading experience
- Simple 4-character key system for file transfer
- Automatic file cleanup after timeout
- Minimal Docker image (~15MB)

## Quick Start

### Docker

```bash
# Build and run with Docker Compose
docker compose up -d

# Or build manually
docker build -t epub2kobo .
docker run -p 3001:3001 epub2kobo
```

### Local Development

```bash
# Build the Go binary
go build -o epub2kobo

# Run the server
./epub2kobo
```

The service runs on http://localhost:3001

## Usage

1. **On your Kobo:** Open the browser and navigate to the service URL
2. **On your phone/computer:** Enter the 4-character key shown on Kobo
3. Upload your EPUB file (max 800MB)
4. The file downloads automatically to your Kobo

Files are automatically deleted after 30 seconds of inactivity or 1 hour maximum.

## Configuration

- **Port:** 3001 (configurable in docker-compose.yaml)
- **Max file size:** 800MB
- **Supported formats:** EPUB only
- **Storage:** Temporary (files auto-delete after timeout)

## Optional Dependencies

- [kepubify](https://github.com/pgaskin/kepubify) - For KEPUB conversion (included in Docker image)

## License

MIT
