package appnet

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppManifest describes an installable app.
// Stored as app.json in each app's directory.
type AppManifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Icon        string            `json:"icon"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Command     string            `json:"command"`     // e.g., "python3 server.py" or "./app"
	Port        int               `json:"port"`        // port the app listens on inside namespace
	Category    string            `json:"category"`    // core, productivity, media, developer, system
	Keywords    []string          `json:"keywords"`
	Env         map[string]string `json:"env"`         // extra env vars
	Deps        []string          `json:"deps"`        // alpine packages needed
	WorkDir     string            `json:"work_dir"`    // defaults to app directory
	AutoStart   bool              `json:"auto_start"`  // start on boot
	Singleton   bool              `json:"singleton"`   // only one instance allowed
}

// LoadManifest reads an app.json file.
func LoadManifest(path string) (*AppManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m AppManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ScanApps finds all app.json manifests in a directory.
// Expected layout: appsDir/calculator/app.json, appsDir/browser/app.json, etc.
func ScanApps(appsDir string) ([]*AppManifest, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, err
	}

	var manifests []*AppManifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(appsDir, e.Name(), "app.json")
		m, err := LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		if m.WorkDir == "" {
			m.WorkDir = filepath.Join(appsDir, e.Name())
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// EnvSlice converts the manifest's env map to a slice for exec.
func (m *AppManifest) EnvSlice() []string {
	var env []string
	for k, v := range m.Env {
		env = append(env, k+"="+v)
	}
	return env
}
