package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSavedNoteCreatesItem(t *testing.T) {
	dir := t.TempDir()
	sc, _ := loadSavedCrypto(dir)
	s := &Server{dataDir: dir, savedCrypto: sc}

	req := httptest.NewRequest("POST", "/api/saved/note", strings.NewReader(`{"text":"  buy milk  "}`))
	w := httptest.NewRecorder()
	s.handleSavedNote(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var out savedItemOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Kind != "note" || out.Text != "buy milk" || out.Domain != "" || out.SavedAt == 0 {
		t.Fatalf("unexpected note item: %+v", out)
	}
	if !out.Available || out.IsActive {
		t.Fatalf("note should be available and not config-active: %+v", out)
	}
	if s.savedCount() != 1 {
		t.Fatalf("count = %d, want 1", s.savedCount())
	}
}

func TestSavedLockUnlockFlow(t *testing.T) {
	dir := t.TempDir()
	sc, _ := loadSavedCrypto(dir)
	s := &Server{dataDir: dir, savedCrypto: sc}

	// Save a note, then set a passphrase.
	noteReq := httptest.NewRequest("POST", "/api/saved/note", strings.NewReader(`{"text":"secret"}`))
	s.handleSavedNote(httptest.NewRecorder(), noteReq)
	lockReq := httptest.NewRequest("POST", "/api/saved/lock", strings.NewReader(`{"passphrase":"pw"}`))
	lw := httptest.NewRecorder()
	s.handleSavedLock(lw, lockReq)
	if lw.Code != 200 {
		t.Fatalf("set passphrase = %d", lw.Code)
	}

	// Simulate a restart: reload crypto -> locked.
	s.savedCrypto, _ = loadSavedCrypto(dir)
	if !s.savedCrypto.locked {
		t.Fatal("store should be locked after reload")
	}
	lr := httptest.NewRecorder()
	s.handleSavedList(lr, httptest.NewRequest("GET", "/api/saved", nil))
	if lr.Code != http.StatusLocked {
		t.Fatalf("locked list = %d, want 423", lr.Code)
	}

	// Wrong then right passphrase.
	bw := httptest.NewRecorder()
	s.handleSavedUnlock(bw, httptest.NewRequest("POST", "/api/saved/unlock", strings.NewReader(`{"passphrase":"nope"}`)))
	if bw.Code != http.StatusUnauthorized {
		t.Fatalf("wrong unlock = %d, want 401", bw.Code)
	}
	gw := httptest.NewRecorder()
	s.handleSavedUnlock(gw, httptest.NewRequest("POST", "/api/saved/unlock", strings.NewReader(`{"passphrase":"pw"}`)))
	if gw.Code != 200 || s.savedCrypto.locked {
		t.Fatalf("unlock failed: code=%d locked=%v", gw.Code, s.savedCrypto.locked)
	}
	if s.savedList()[0].Text != "secret" {
		t.Fatal("data not readable after unlock")
	}
}

func TestHandleSavedFromChatSnapshot(t *testing.T) {
	dir := t.TempDir()
	sc, _ := loadSavedCrypto(dir)
	s := &Server{dataDir: dir, savedCrypto: sc}

	req := httptest.NewRequest("POST", "/api/saved/from-chat",
		strings.NewReader(`{"text":"see you at 8","contactName":"Sara"}`))
	w := httptest.NewRecorder()
	s.handleSavedFromChat(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	var out savedItemOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Kind != "chat" || out.Text != "see you at 8" || out.Nickname != "Sara" || out.Domain != "" {
		t.Fatalf("unexpected chat item: %+v", out)
	}
	if out.ConfigLabel != "Sara" || !out.Available {
		t.Fatalf("chat item should label by contact and be available: %+v", out)
	}
	// Deleting the snapshot is a pure store op (no chat hub involved).
	if removed, err := s.savedDeleteAndCleanup(out.ID); err != nil || removed == nil {
		t.Fatalf("delete failed: %v %+v", err, removed)
	}
	if s.savedCount() != 0 {
		t.Fatalf("count after delete = %d, want 0", s.savedCount())
	}
}

func TestHandleSavedNoteRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	sc, _ := loadSavedCrypto(dir)
	s := &Server{dataDir: dir, savedCrypto: sc}
	req := httptest.NewRequest("POST", "/api/saved/note", strings.NewReader(`{"text":"   "}`))
	w := httptest.NewRecorder()
	s.handleSavedNote(w, req)
	if w.Code != 400 {
		t.Fatalf("empty note status = %d, want 400", w.Code)
	}
	if s.savedCount() != 0 {
		t.Fatalf("empty note was stored: count=%d", s.savedCount())
	}
}
