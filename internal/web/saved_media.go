package web

import (
	"net/http"
	"strconv"
	"strings"
)

// persistSavedMedia copies (size, crc) bytes from the ephemeral media cache
// into the never-reaped saved-media store. Returns true if the bytes are now
// in saved-media (including the already-present case), false on cache miss.
func (s *Server) persistSavedMedia(size int64, crc uint32) bool {
	if s.savedMedia == nil || s.mediaCache == nil || size <= 0 || crc == 0 {
		return false
	}
	if _, _, ok := s.savedMedia.Get(size, crc); ok {
		return true // already persisted
	}
	body, mime, ok := s.mediaCache.Get(size, crc)
	if !ok {
		return false
	}
	return s.savedMedia.Put(size, crc, body, mime) == nil
}

// handleSavedMedia serves persisted media by ?size=&crc=, falling back to the
// ephemeral cache if the bytes happen to still be there.
func (s *Server) handleSavedMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	size, _ := strconv.ParseInt(q.Get("size"), 10, 64)
	crcU, err := strconv.ParseUint(strings.TrimSpace(q.Get("crc")), 16, 32)
	if err != nil || size <= 0 {
		http.Error(w, "bad size/crc", http.StatusBadRequest)
		return
	}
	crc := uint32(crcU)
	var body []byte
	var mime string
	var ok bool
	if s.savedMedia != nil {
		body, mime, ok = s.savedMedia.Get(size, crc)
	}
	if !ok && s.mediaCache != nil {
		body, mime, ok = s.mediaCache.Get(size, crc)
	}
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if mime == "" {
		mime = http.DetectContentType(body)
	}
	w.Header().Set("Content-Type", sanitizeMime(mime))
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	// Defence in depth: never let a browser MIME-sniff this user-controlled blob,
	// and force download-style handling rather than inline rendering.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	_, _ = w.Write(body)
}
