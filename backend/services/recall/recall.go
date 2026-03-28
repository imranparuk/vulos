package recall

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vulos/backend/internal/vecdb"
	"vulos/backend/services/embeddings"
)

// Recall is the semantic search service.
// It indexes files from the filesystem into a vector database,
// so users can ask "show me that hex logo from last Tuesday"
// instead of browsing directories.
type Recall struct {
	db       *vecdb.DB
	embedder *embeddings.Embedder
	dataDir  string
	mu       sync.Mutex
	status   IndexStatus
}

type IndexStatus struct {
	TotalFiles   int       `json:"total_files"`
	IndexedFiles int       `json:"indexed_files"`
	LastIndex    time.Time `json:"last_index,omitempty"`
	Indexing     bool      `json:"indexing"`
	Error        string    `json:"error,omitempty"`
}

func New(dbPath, dataDir string, embedder *embeddings.Embedder) (*Recall, error) {
	db, err := vecdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open vecdb: %w", err)
	}
	return &Recall{
		db:       db,
		embedder: embedder,
		dataDir:  dataDir,
	}, nil
}

// Search performs semantic search across indexed files.
func (r *Recall) Search(ctx context.Context, query string, topK int) ([]vecdb.SearchResult, error) {
	emb, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results := r.db.Search(emb, topK)
	return results, nil
}

// IndexAll walks the data directory and indexes every supported file.
func (r *Recall) IndexAll(ctx context.Context) error {
	r.mu.Lock()
	if r.status.Indexing {
		r.mu.Unlock()
		return fmt.Errorf("indexing already in progress")
	}
	r.status.Indexing = true
	r.status.Error = ""
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.status.Indexing = false
		r.status.LastIndex = time.Now()
		r.mu.Unlock()
		r.db.Flush()
	}()

	var total, indexed int

	err := filepath.WalkDir(r.dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		if !isSupportedFile(path) {
			return nil
		}

		total++
		docID := fileID(path)

		// Skip if already indexed (by hash)
		if _, exists := r.db.Get(docID); exists {
			indexed++
			return nil
		}

		content, err := readFileContent(path)
		if err != nil || content == "" {
			return nil
		}

		emb, err := r.embedder.Embed(ctx, content)
		if err != nil {
			log.Printf("[recall] embed error for %s: %v", path, err)
			return nil
		}

		relPath, _ := filepath.Rel(r.dataDir, path)
		info, _ := d.Info()
		modTime := ""
		if info != nil {
			modTime = info.ModTime().Format(time.RFC3339)
		}

		r.db.Upsert(&vecdb.Document{
			ID:        docID,
			Content:   truncate(content, 500),
			Embedding: emb,
			Metadata: map[string]string{
				"path":     relPath,
				"abs_path": path,
				"modified": modTime,
				"ext":      filepath.Ext(path),
			},
		})
		indexed++
		return nil
	})

	r.mu.Lock()
	r.status.TotalFiles = total
	r.status.IndexedFiles = indexed
	if err != nil {
		r.status.Error = err.Error()
	}
	r.mu.Unlock()

	log.Printf("[recall] indexed %d/%d files", indexed, total)
	return err
}

// Status returns current index status.
func (r *Recall) Status() IndexStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.status
	s.IndexedFiles = r.db.Count()
	return s
}

// StartSchedule re-indexes on an interval.
func (r *Recall) StartSchedule(ctx context.Context, interval time.Duration) {
	go func() {
		// Initial index
		if err := r.IndexAll(ctx); err != nil {
			log.Printf("[recall] initial index error: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.IndexAll(ctx); err != nil {
					log.Printf("[recall] scheduled index error: %v", err)
				}
			}
		}
	}()
}

func fileID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", h[:12])
}

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supported := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".go": true, ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
		".py": true, ".rs": true, ".html": true, ".css": true, ".sql": true,
		".sh": true, ".toml": true, ".cfg": true, ".ini": true, ".env": true,
		".csv": true, ".xml": true, ".svg": true, ".log": true,
	}
	return supported[ext]
}

func readFileContent(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	// Skip files > 1MB
	if info.Size() > 1<<20 {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
