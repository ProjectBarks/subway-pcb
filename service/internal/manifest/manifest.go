// Package manifest reads a Vite manifest.json and resolves logical asset
// names (e.g. "main.css") to their content-hashed paths (e.g.
// "/static/dist/main-abc123.css").
package manifest

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	File string   `json:"file"`
	CSS  []string `json:"css,omitempty"`
}

// byBase maps unhashed base names ("main.css") to hashed filenames ("main-abc123.css").
var byBase map[string]string

// Load reads the Vite manifest from distDir/.vite/manifest.json.
// It is safe to call with a missing manifest (e.g. dev mode); lookups
// will fall back to unhashed paths.
func Load(distDir string) error {
	p := filepath.Join(distDir, ".vite", "manifest.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var raw map[string]entry
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m := make(map[string]string, len(raw)*2)
	for _, e := range raw {
		base := stripHash(e.File)
		m[base] = e.File
		for _, css := range e.CSS {
			m[stripHash(css)] = css
		}
	}
	byBase = m
	log.Printf("manifest: loaded %d assets from %s", len(m), p)
	return nil
}

// Asset resolves a base name like "main.css" to "/static/dist/main-abc123.css".
// Falls back to the unhashed path if the manifest was not loaded.
func Asset(name string) string {
	if byBase != nil {
		if hashed, ok := byBase[name]; ok {
			return "/static/dist/" + hashed
		}
	}
	return "/static/dist/" + name
}

// stripHash converts "main-abc123.css" to "main.css".
func stripHash(filename string) string {
	ext := filepath.Ext(filename)
	stem := strings.TrimSuffix(filename, ext)
	if i := strings.LastIndex(stem, "-"); i >= 0 {
		return stem[:i] + ext
	}
	return filename
}
