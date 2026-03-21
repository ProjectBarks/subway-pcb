package plugin

import "github.com/ProjectBarks/subway-pcb/service/internal/model"

// Plugin defines a pluggable rendering strategy for LED output.
type Plugin interface {
	Name() string
	Description() string
	RequiredFeatures() []string
	ConfigFields() []ConfigField
	DefaultPresets() []model.Preset
	LuaSource() string
}

// IsPluginCompatible returns true if the board has all features required by the plugin.
func IsPluginCompatible(requiredFeatures, boardFeatures []string) bool {
	if len(requiredFeatures) == 0 {
		return true
	}
	featureSet := make(map[string]bool, len(boardFeatures))
	for _, f := range boardFeatures {
		featureSet[f] = true
	}
	for _, f := range requiredFeatures {
		if !featureSet[f] {
			return false
		}
	}
	return true
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
