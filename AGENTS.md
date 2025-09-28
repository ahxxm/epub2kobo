# Send2Ereader Agent Guidelines

## Build/Test Commands
- **Start server:** `npm start` (runs on port 3001)
- **Install deps:** `npm install` (runs patch-package postinstall)
- **Docker build:** `docker compose build`
- **Docker run:** `docker compose up -d`
- **Testing:** No test suite - verify manually at http://localhost:3001

## Service Behavior & Data Flow
Kobo browser workflow: User opens `/` → generates 4-char key → polls `/status/{key}` every 5s → downloads file via `/{filename}?key={key}`.
Desktop/phone workflow: User enters key → uploads EPUB to `/upload` → optionally converted to KEPUB → stored in `uploads/` → served to Kobo.
**Security:** Keys expire 30s after last activity, max 1hr lifetime. User-agent validation prevents key hijacking. Files auto-delete on expiry.
**Conversions:** EPUB→KEPUB for Kobo (kepubify) - optional conversion for better reading experience.
**File limits:** 800MB max, allowed: .epub only

## Code Patterns
- **Framework:** Koa with @koa/router, multer for uploads, koa-static for static files
- **Style:** CommonJS (`require`/`module.exports`), async/await for routes, camelCase vars, UPPER_CASE consts
- **Error handling:** Flash messages via `flash()` function, cleanup files on error, console.error for logging
- **File ops:** Always use absolute paths with path.resolve/join, sanitize-filename for user input, transliteration for non-ASCII
