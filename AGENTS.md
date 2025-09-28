# Send2Ereader Agent Guidelines

## Build/Test Commands
- **Build binary:** `go build -o epub2kobo main.go`
- **Run server:** `./epub2kobo` or `go run main.go` (runs on port 3001)
- **Docker build:** `docker compose build` or `DOCKER_BUILDKIT=1 docker build -t epub2kobo .`
- **Docker run:** `docker compose up -d`
- **Testing:** No test suite - requires network-accessible server (e.g., `http://192.168.1.x:3001`). Kobo and upload device must both use server's network IP address

## Service Behavior & Data Flow
Kobo browser workflow: User opens `/` → generates 4-char key → polls `/status/{key}` every 5s → downloads file via `/{filename}?key={key}`.
Desktop/phone workflow: User enters key → uploads EPUB to `/upload` → optionally converted to KEPUB → stored in `uploads/` directory → served to Kobo.
**Security:** Keys expire 30s after last activity, max 1hr lifetime. User-agent validation prevents key hijacking. Files auto-delete on expiry.
**Conversions:** EPUB→KEPUB for Kobo (kepubify) - optional conversion for better reading experience.
**File limits:** 800MB max, allowed: .epub only

## Code Patterns
- **Language:** Pure Go with standard library only (net/http, sync.Map, html/template, embed)
- **Architecture:** Single-binary service with embedded static files via `//go:embed`
- **Style:** Go conventions - camelCase functions/vars, error checking after operations, defer for cleanup
- **Concurrency:** sync.Map for thread-safe key storage, goroutines for TTL cleanup
- **Error handling:** HTTP error responses with proper status codes, log.Printf for logging
- **File ops:** In-memory storage (no filesystem persistence), sanitized filenames, temp files for kepubify conversion
