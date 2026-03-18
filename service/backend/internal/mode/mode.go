package mode

import "github.com/ProjectBarks/subway-pcb/server/internal/model"

// Mode defines a pluggable rendering strategy for LED output.
type Mode interface {
	Name() string
	Description() string
	ConfigFields() []ConfigField
	DefaultThemes() []model.Theme
	Render(ctx RenderContext) ([]byte, error)
}

// Registry holds all registered modes in insertion order.
type Registry struct {
	modes map[string]Mode
	order []string
}

func NewRegistry() *Registry {
	return &Registry{modes: make(map[string]Mode)}
}

func (r *Registry) Register(m Mode) {
	if _, exists := r.modes[m.Name()]; !exists {
		r.order = append(r.order, m.Name())
	}
	r.modes[m.Name()] = m
}

func (r *Registry) Get(name string) (Mode, bool) {
	m, ok := r.modes[name]
	return m, ok
}

func (r *Registry) List() []Mode {
	modes := make([]Mode, 0, len(r.order))
	for _, name := range r.order {
		if m, ok := r.modes[name]; ok {
			modes = append(modes, m)
		}
	}
	return modes
}

// AllDefaultThemes returns built-in themes from all registered modes.
func (r *Registry) AllDefaultThemes() []model.Theme {
	var all []model.Theme
	for _, name := range r.order {
		all = append(all, r.modes[name].DefaultThemes()...)
	}
	return all
}
