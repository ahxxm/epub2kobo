# Send2Ereader Agent Guidelines

## Build/Test Commands
- **Start server:** `npm start` (runs on port 3001)
- **Install deps:** `npm install` (runs patch-package postinstall)
- **Docker build:** `docker compose build`
- **Docker run:** `docker compose up -d`
- **Testing:** No test suite - verify manually at http://localhost:3001

## Service Behavior & Data Flow
E-reader browser workflow: User opens `/` → generates 4-char key → polls `/status/{key}` every 5s → downloads file via `/{filename}?key={key}`.
Desktop/phone workflow: User enters key → uploads ebook to `/upload` → file converted if needed → stored in `uploads/` → served to e-reader.
**Security:** Keys expire 30s after last activity, max 1hr lifetime. User-agent validation prevents key hijacking. Files auto-delete on expiry.
**Conversions:** EPUB→MOBI for Kindle (kindlegen), EPUB→KEPUB for Kobo (kepubify), PDF margin cropping (pdfcropmargins).
**File limits:** 800MB max, allowed: .epub/.mobi/.pdf/.cbz/.cbr/.html/.txt

## Code Patterns
- **Framework:** Koa with @koa/router, multer for uploads, koa-static for static files
- **Style:** CommonJS (`require`/`module.exports`), async/await for routes, camelCase vars, UPPER_CASE consts
- **Error handling:** Flash messages via `flash()` function, cleanup files on error, console.error for logging
- **File ops:** Always use absolute paths with path.resolve/join, sanitize-filename for user input, transliteration for non-ASCII
