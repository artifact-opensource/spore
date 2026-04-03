package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/artifact-virtual/symbiote-android/provider"
)

// Session represents a persistent conversation
type Session struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Created   time.Time          `json:"created"`
	Updated   time.Time          `json:"updated"`
	Messages  []provider.Message `json:"messages"`
}

// SessionMeta is the lightweight version for listing
type SessionMeta struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Count   int       `json:"message_count"`
}

// SessionStore manages persistent sessions on disk
type SessionStore struct {
	dir string
	mu  sync.RWMutex
}

func NewSessionStore(dataDir string) *SessionStore {
	dir := filepath.Join(dataDir, "sessions")
	os.MkdirAll(dir, 0755)
	return &SessionStore{dir: dir}
}

func (s *SessionStore) sessionPath(id string) string {
	// Sanitize ID to prevent directory traversal
	safe := strings.ReplaceAll(id, "/", "")
	safe = strings.ReplaceAll(safe, "\\", "")
	safe = strings.ReplaceAll(safe, "..", "")
	return filepath.Join(s.dir, safe+".json")
}

// Create a new session, returns it
func (s *SessionStore) Create(title string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	id := fmt.Sprintf("s_%d", now.UnixMilli())

	if title == "" {
		title = "New Session"
	}

	sess := &Session{
		ID:       id,
		Title:    title,
		Created:  now,
		Updated:  now,
		Messages: []provider.Message{},
	}

	s.writeLocked(sess)
	return sess
}

// Load a session by ID
func (s *SessionStore) Load(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.sessionPath(id))
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("corrupt session: %s", id)
	}
	return &sess, nil
}

// Save a session (full overwrite)
func (s *SessionStore) Save(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Updated = time.Now()
	return s.writeLocked(sess)
}

func (s *SessionStore) writeLocked(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sessionPath(sess.ID), data, 0644)
}

// Delete a session
func (s *SessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.sessionPath(id))
}

// List all sessions (sorted by updated, newest first)
func (s *SessionStore) List() []SessionMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}

	var metas []SessionMeta
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}

		// Count only user+assistant messages for display
		count := 0
		for _, m := range sess.Messages {
			if m.Role == "user" || m.Role == "assistant" {
				count++
			}
		}

		metas = append(metas, SessionMeta{
			ID:      sess.ID,
			Title:   sess.Title,
			Created: sess.Created,
			Updated: sess.Updated,
			Count:   count,
		})
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Updated.After(metas[j].Updated)
	})

	return metas
}

// UpdateTitle auto-generates title from first user message if still default
func (s *SessionStore) UpdateTitle(sess *Session) {
	if sess.Title != "New Session" {
		return
	}
	for _, m := range sess.Messages {
		if m.Role == "user" && m.Content != "" {
			title := m.Content
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			sess.Title = title
			return
		}
	}
}
