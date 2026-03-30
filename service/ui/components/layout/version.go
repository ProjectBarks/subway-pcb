package layout

import "github.com/ProjectBarks/subway-pcb/service/internal/manifest"

// Asset resolves a base asset name (e.g. "main.css") to its full
// content-hashed path (e.g. "/static/dist/main-abc123.css").
func Asset(name string) string {
	return manifest.Asset(name)
}
