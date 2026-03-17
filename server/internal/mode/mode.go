package mode

import (
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
)

// RenderContext provides everything a mode needs to produce a pixel frame.
type RenderContext struct {
	Aggregator *mta.Aggregator
	StationIDs []string // flat LED index -> station ID (from LEDMap)
	Theme      *model.Theme
	Device     *model.Device
	TotalLEDs  int
}

// Mode defines a pluggable rendering strategy for LED output.
type Mode interface {
	Name() string
	Description() string
	Render(ctx RenderContext) ([]byte, error) // returns RGB byte array (len = TotalLEDs*3)
}

// Registry holds all registered modes.
type Registry struct {
	modes map[string]Mode
}

// NewRegistry creates an empty mode registry.
func NewRegistry() *Registry {
	return &Registry{modes: make(map[string]Mode)}
}

// Register adds a mode to the registry.
func (r *Registry) Register(m Mode) {
	r.modes[m.Name()] = m
}

// Get returns a mode by name, or nil if not found.
func (r *Registry) Get(name string) (Mode, bool) {
	m, ok := r.modes[name]
	return m, ok
}

// List returns all registered modes.
func (r *Registry) List() []Mode {
	modes := make([]Mode, 0, len(r.modes))
	for _, m := range r.modes {
		modes = append(modes, m)
	}
	return modes
}
