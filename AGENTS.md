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

## Agentic Testing Commands
For AI agents to test the complete workflow:
```bash
# Start server
./epub2kobo &
SERVER_PID=$!
sleep 1

# 1. Generate key (Kobo side)
KEY=$(curl -s -X POST -H "User-Agent: Mozilla/5.0 (Kobo Touch)" http://localhost:3001/generate | jq -r .key)
echo "Generated key: $KEY"

# 2. Verify key not ready
curl -s http://localhost:3001/status/$KEY | jq '.ready' # should be false

# 3. Create minimal test EPUB
mkdir -p test_epub/META-INF
echo 'application/epub+zip' > test_epub/mimetype
echo '<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>' > test_epub/META-INF/container.xml
echo '<?xml version="1.0"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Test Book</dc:title></metadata>
</package>' > test_epub/content.opf
cd test_epub && zip -0 -X ../test.epub mimetype && zip -r ../test.epub . -x mimetype && cd ..

# 4. Upload file (Desktop side)
RESPONSE=$(curl -s -X POST -F "key=$KEY" -F "file=@test.epub" -F "kepubify=on" http://localhost:3001/upload)
echo "Upload response: $RESPONSE"

# 5. Verify file ready
STATUS=$(curl -s http://localhost:3001/status/$KEY)
echo "Status: $(echo $STATUS | jq '.ready')" # should be true
FILENAME=$(echo $STATUS | jq -r .filename)

# 6. Test download (Kobo side)
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:3001/$FILENAME?key=$KEY")
echo "Download HTTP status: $HTTP_STATUS" # should be 200

# Cleanup
rm -rf test_epub test.epub
kill $SERVER_PID
```
