package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Conversation is a saved chat session.
type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HistoryStore persists conversation history per user.
type HistoryStore struct {
	mu   sync.RWMutex
	path string
	data map[string][]*Conversation // userID → conversations
}

func NewHistoryStore(dataDir string) *HistoryStore {
	p := filepath.Join(dataDir, "chat_history.json")
	h := &HistoryStore{
		path: p,
		data: make(map[string][]*Conversation),
	}
	if raw, err := os.ReadFile(p); err == nil {
		json.Unmarshal(raw, &h.data)
	}
	return h
}

// Save stores a message exchange in the active conversation for a user.
func (h *HistoryStore) Save(userID string, msgs []Message) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	convs := h.data[userID]
	// Append to last conversation if recent, otherwise create new
	var conv *Conversation
	if len(convs) > 0 {
		last := convs[len(convs)-1]
		if time.Since(last.UpdatedAt) < 30*time.Minute {
			conv = last
		}
	}
	if conv == nil {
		title := "Chat"
		if len(msgs) > 0 {
			t := msgs[0].Content
			if len(t) > 50 {
				t = t[:50] + "..."
			}
			title = t
		}
		conv = &Conversation{
			ID:        time.Now().Format("20060102150405.000"),
			UserID:    userID,
			Title:     title,
			CreatedAt: time.Now(),
		}
		h.data[userID] = append(convs, conv)
	}

	conv.Messages = append(conv.Messages, msgs...)
	conv.UpdatedAt = time.Now()

	// Keep max 50 conversations per user
	if len(h.data[userID]) > 50 {
		h.data[userID] = h.data[userID][len(h.data[userID])-50:]
	}

	return conv.ID
}

// List returns conversations for a user, newest first.
func (h *HistoryStore) List(userID string, limit int) []*Conversation {
	h.mu.RLock()
	defer h.mu.RUnlock()

	convs := h.data[userID]
	if limit <= 0 || limit > len(convs) {
		limit = len(convs)
	}
	result := make([]*Conversation, limit)
	for i := 0; i < limit; i++ {
		result[i] = convs[len(convs)-1-i]
	}
	return result
}

// Get returns a specific conversation.
func (h *HistoryStore) Get(userID, convID string) *Conversation {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.data[userID] {
		if c.ID == convID {
			return c
		}
	}
	return nil
}

// Delete removes a conversation.
func (h *HistoryStore) Delete(userID, convID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	convs := h.data[userID]
	for i, c := range convs {
		if c.ID == convID {
			h.data[userID] = append(convs[:i], convs[i+1:]...)
			return
		}
	}
}

// Flush persists to disk.
func (h *HistoryStore) Flush() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	raw, err := json.Marshal(h.data)
	if err != nil {
		return err
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, h.path)
}
