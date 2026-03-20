package plugin

import "github.com/ProjectBarks/subway-pcb/service/internal/model"

// Plugin defines a pluggable rendering strategy for LED output.
type Plugin interface {
	Name() string
	Description() string
	ConfigFields() []ConfigField
	DefaultPresets() []model.Preset
	Render(ctx RenderContext) ([]byte, error)
}

// Registry holds all registered plugins in insertion order.
type Registry struct {
	plugins map[string]Plugin
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) Register(p Plugin) {
	if _, exists := r.plugins[p.Name()]; !exists {
		r.order = append(r.order, p.Name())
	}
	r.plugins[p.Name()] = p
}

func (r *Registry) Get(name string) (Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []Plugin {
	plugins := make([]Plugin, 0, len(r.order))
	for _, name := range r.order {
		if p, ok := r.plugins[name]; ok {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// AllDefaultPresets returns built-in presets from all registered plugins.
func (r *Registry) AllDefaultPresets() []model.Preset {
	var all []model.Preset
	for _, name := range r.order {
		all = append(all, r.plugins[name].DefaultPresets()...)
	}
	return all
}
