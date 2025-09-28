package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

//go:embed static/upload.html static/download.html static/style.css
var staticFiles embed.FS

const (
	maxUploadSize = 800 * 1024 * 1024 // 800MB
	keyLength     = 4
	keyTimeout    = 30 * time.Second
	maxKeyAge     = 1 * time.Hour
)

type FileEntry struct {
	Key       string
	Filename  string
	Path      string
	Converted bool
	CreatedAt time.Time
	LastSeen  time.Time
	mu        sync.Mutex
}

type Server struct {
	store      sync.Map // map[string]*FileEntry
	uploadsDir string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	server := &Server{
		uploadsDir: "uploads",
	}

	// Create uploads directory
	if err := os.MkdirAll(server.uploadsDir, 0755); err != nil {
		log.Fatal("Failed to create uploads directory:", err)
	}

	// Cleanup routine
	go server.cleanupRoutine()

	// Setup routes
	http.HandleFunc("/", server.handleRoot)
	http.HandleFunc("/upload", server.handleUpload)
	http.HandleFunc("/status/", server.handleStatus)
	http.HandleFunc("/generate", server.handleGenerate)
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// File download route (matches any filename)
	http.HandleFunc("/{filename}", server.handleDownload)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		server.cleanupAllFiles()
		os.Exit(0)
	}()

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// Try to serve file download
		s.handleDownload(w, r)
		return
	}

	// Check if Kobo browser
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(userAgent, "Kobo") {
		// Serve download page
		content, err := staticFiles.ReadFile("static/download.html")
		if err != nil {
			http.Error(w, "Page not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	} else {
		// Redirect to upload page
		http.Redirect(w, r, "/static/upload.html", http.StatusFound)
	}
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := generateKey()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with size limit
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too large or invalid", http.StatusBadRequest)
		return
	}

	key := r.FormValue("key")
	if len(key) != keyLength || !isValidKey(key) {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate EPUB mime type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, "Cannot read file", http.StatusBadRequest)
		return
	}
	file.Seek(0, 0)

	mimeType := http.DetectContentType(buffer)
	if !strings.HasPrefix(mimeType, "application/epub+zip") && !strings.HasPrefix(mimeType, "application/zip") {
		// Additional check for EPUB signature
		if !isEPUB(file) {
			http.Error(w, "Only EPUB files are allowed", http.StatusBadRequest)
			return
		}
		file.Seek(0, 0)
	}

	// Sanitize filename
	filename := sanitizeFilename(header.Filename)
	if !strings.HasSuffix(strings.ToLower(filename), ".epub") {
		filename += ".epub"
	}

	// Save uploaded file
	tempPath := filepath.Join(s.uploadsDir, fmt.Sprintf("%s_%s", key, filename))
	destFile, err := os.Create(tempPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		os.Remove(tempPath)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	finalPath := tempPath
	converted := false

	// Check if kepubify conversion requested
	if r.FormValue("kepubify") == "on" {
		if kepubPath := convertToKepub(tempPath); kepubPath != "" {
			os.Remove(tempPath)
			finalPath = kepubPath
			converted = true
			filename = strings.TrimSuffix(filename, ".epub") + ".kepub.epub"
		}
	}

	// Store file entry
	entry := &FileEntry{
		Key:       key,
		Filename:  filename,
		Path:      finalPath,
		Converted: converted,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	s.store.Store(key, entry)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"filename": filename,
		"key":      key,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/status/")
	if len(key) != keyLength {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}

	value, exists := s.store.Load(key)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": "Key not found or expired"})
		return
	}

	entry := value.(*FileEntry)
	entry.mu.Lock()
	entry.LastSeen = time.Now()
	entry.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":     true,
		"filename":  entry.Filename,
		"converted": entry.Converted,
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/")
	if filename == "" {
		http.NotFound(w, r)
		return
	}

	key := r.URL.Query().Get("key")
	if len(key) != keyLength {
		http.NotFound(w, r)
		return
	}

	value, exists := s.store.Load(key)
	if !exists {
		http.NotFound(w, r)
		return
	}

	entry := value.(*FileEntry)
	
	// Update last seen
	entry.mu.Lock()
	entry.LastSeen = time.Now()
	entry.mu.Unlock()

	// Verify filename matches
	if entry.Filename != filename {
		http.NotFound(w, r)
		return
	}

	// Serve file
	file, err := os.Open(entry.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/epub+zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", entry.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	io.Copy(w, file)
}

func (s *Server) cleanupRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.store.Range(func(key, value interface{}) bool {
			entry := value.(*FileEntry)
			entry.mu.Lock()
			lastSeen := entry.LastSeen
			createdAt := entry.CreatedAt
			entry.mu.Unlock()

			// Remove if inactive for 30s or older than 1 hour
			if now.Sub(lastSeen) > keyTimeout || now.Sub(createdAt) > maxKeyAge {
				s.store.Delete(key)
				os.Remove(entry.Path)
				log.Printf("Cleaned up file: %s (key: %s)", entry.Filename, entry.Key)
			}
			return true
		})
	}
}

func (s *Server) cleanupAllFiles() {
	s.store.Range(func(key, value interface{}) bool {
		entry := value.(*FileEntry)
		os.Remove(entry.Path)
		s.store.Delete(key)
		return true
	})
}

func generateKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, keyLength)
	rand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

func isValidKey(key string) bool {
	match, _ := regexp.MatchString("^[a-z0-9]+$", key)
	return match
}

func sanitizeFilename(filename string) string {
	// Remove path components
	filename = filepath.Base(filename)
	
	// Replace non-ASCII with transliteration
	var result []rune
	for _, r := range filename {
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || r == ' ') {
			result = append(result, r)
		} else if r == '–' || r == '—' {
			result = append(result, '-')
		} else if unicode.IsSpace(r) {
			result = append(result, ' ')
		}
	}
	
	sanitized := string(result)
	
	// Remove multiple spaces/dots
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	sanitized = regexp.MustCompile(`\.+`).ReplaceAllString(sanitized, ".")
	sanitized = strings.TrimSpace(sanitized)
	
	if sanitized == "" {
		sanitized = "book"
	}
	
	return sanitized
}

func isEPUB(file multipart.File) bool {
	// Check for EPUB signature (PK files with mimetype as first file)
	buffer := make([]byte, 58)
	n, err := file.Read(buffer)
	if err != nil || n < 58 {
		return false
	}
	
	// Check PK signature and "mimetype" string
	if buffer[0] == 0x50 && buffer[1] == 0x4B && // PK
		string(buffer[30:38]) == "mimetype" &&
		string(buffer[38:58]) == "application/epub+zip" {
		return true
	}
	
	return false
}

func convertToKepub(epubPath string) string {
	// Check if kepubify exists
	if _, err := exec.LookPath("kepubify"); err != nil {
		return ""
	}
	
	kepubPath := strings.TrimSuffix(epubPath, ".epub") + ".kepub.epub"
	cmd := exec.Command("kepubify", "-o", kepubPath, epubPath)
	
	if err := cmd.Run(); err != nil {
		log.Printf("kepubify conversion failed: %v", err)
		return ""
	}
	
	return kepubPath
}
