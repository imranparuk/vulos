package pty

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/creack/pty"
	"golang.org/x/net/websocket"
)

// Service manages PTY sessions bridged to WebSocket clients.
type Service struct {
	mu       sync.Mutex
	sessions map[string]*Session
	shell    string
}

// Session is a single PTY + WebSocket bridge.
type Session struct {
	ID     string
	UserID string
	ptmx   *os.File
	cmd    *exec.Cmd
	done   chan struct{}
}

func NewService() *Service {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return &Service{
		sessions: make(map[string]*Session),
		shell:    shell,
	}
}

// Handler returns an http.Handler for the WebSocket PTY endpoint.
// Connect via: ws://host:port/api/pty?cols=80&rows=24
func (s *Service) Handler() http.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame

		userID := ws.Request().Header.Get("X-User-ID")
		query := ws.Request().URL.Query()

		cols := parseIntOr(query.Get("cols"), 80)
		rows := parseIntOr(query.Get("rows"), 24)

		sess, err := s.createSession(userID, uint16(cols), uint16(rows))
		if err != nil {
			log.Printf("[pty] failed to create session: %v", err)
			return
		}
		defer s.destroySession(sess.ID)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// PTY → WebSocket
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := sess.ptmx.Read(buf)
				if err != nil {
					cancel()
					return
				}
				if n > 0 {
					// Ensure valid UTF-8 for WebSocket text frames
					data := sanitizeUTF8(buf[:n])
					if _, err := ws.Write(data); err != nil {
						cancel()
						return
					}
				}
			}
		}()

		// WebSocket → PTY
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := ws.Read(buf)
				if err != nil {
					cancel()
					return
				}
				if n > 0 {
					// Check for resize message: \x01{cols},{rows}
					if buf[0] == 1 && n > 1 {
						s.handleResize(sess, buf[1:n])
						continue
					}
					sess.ptmx.Write(buf[:n])
				}
			}
		}()

		// Wait for context cancellation or process exit
		select {
		case <-ctx.Done():
		case <-sess.done:
		}
	})
}

func (s *Service) createSession(userID string, cols, rows uint16) (*Session, error) {
	cmd := exec.Command(s.shell)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LANG=en_US.UTF-8",
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
	if err != nil {
		return nil, err
	}

	sess := &Session{
		ID:     generateID(),
		UserID: userID,
		ptmx:   ptmx,
		cmd:    cmd,
		done:   make(chan struct{}),
	}

	// Monitor process exit
	go func() {
		cmd.Wait()
		close(sess.done)
	}()

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	log.Printf("[pty] session %s started (user=%s, shell=%s, %dx%d)",
		sess.ID, userID, s.shell, cols, rows)
	return sess, nil
}

func (s *Service) destroySession(id string) {
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	if sess != nil {
		sess.ptmx.Close()
		if sess.cmd.Process != nil {
			sess.cmd.Process.Kill()
		}
		log.Printf("[pty] session %s destroyed", id)
	}
}

func (s *Service) handleResize(sess *Session, data []byte) {
	var cols, rows int
	n, _ := parseTwo(string(data))
	cols = n[0]
	rows = n[1]
	if cols > 0 && rows > 0 {
		pty.Setsize(sess.ptmx, &pty.Winsize{
			Cols: uint16(cols),
			Rows: uint16(rows),
		})
	}
}

// DestroyAll kills all sessions on shutdown.
func (s *Service) DestroyAll() {
	s.mu.Lock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.destroySession(id)
	}
}

// ActiveSessions returns count of active PTY sessions.
func (s *Service) ActiveSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func sanitizeUTF8(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	// Replace invalid bytes
	out := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			out = append(out, '?')
			data = data[1:]
		} else {
			out = append(out, data[:size]...)
			data = data[size:]
		}
	}
	return out
}

func generateID() string {
	return time.Now().Format("20060102150405.000000")
}

func parseIntOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n == 0 {
		return fallback
	}
	return n
}

func parseTwo(s string) ([2]int, bool) {
	var result [2]int
	idx := 0
	for _, c := range s {
		if c == ',' {
			idx = 1
			continue
		}
		if c >= '0' && c <= '9' {
			result[idx] = result[idx]*10 + int(c-'0')
		}
	}
	return result, result[0] > 0 && result[1] > 0
}
