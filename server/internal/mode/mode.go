package mode

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
)

// FieldType controls how the UI renders a config field.
type FieldType string

const (
	FieldColor  FieldType = "color"
	FieldNumber FieldType = "number"
	FieldSelect FieldType = "select"
)

// ConfigField describes a single configurable parameter for a mode.
type ConfigField struct {
	Key     string
	Label   string
	Type    FieldType
	Default string
	Group   string   // UI section grouping
	Min     string   // for number
	Max     string   // for number
	Options []string // for select
}

// RenderContext provides everything a mode needs to produce a pixel frame.
type RenderContext struct {
	Aggregator *mta.Aggregator
	StationIDs []string          // flat LED index -> station ID
	Device     *model.Device
	Config     map[string]string // resolved config (theme defaults + device overrides)
	TotalLEDs  int
}

// ConfigColor reads a hex color from config, falling back to the field default.
func (ctx RenderContext) ConfigColor(key string, fields []ConfigField) (r, g, b uint8) {
	hex := ctx.Config[key]
	if hex == "" {
		for _, f := range fields {
			if f.Key == key {
				hex = f.Default
				break
			}
		}
	}
	if len(hex) == 7 && hex[0] == '#' {
		fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	}
	return
}

// ConfigInt reads a number from config, falling back to the field default.
func (ctx RenderContext) ConfigInt(key string, fields []ConfigField) int {
	s := ctx.Config[key]
	if s == "" {
		for _, f := range fields {
			if f.Key == key {
				s = f.Default
				break
			}
		}
	}
	v, _ := strconv.Atoi(s)
	return v
}

// Mode defines a pluggable rendering strategy for LED output.
type Mode interface {
	Name() string
	Description() string
	ConfigFields() []ConfigField
	DefaultThemes() []model.Theme          // built-in theme presets for this mode
	Render(ctx RenderContext) ([]byte, error)
}

// Registry holds all registered modes.
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

// AllDefaultThemes returns all built-in themes from all registered modes.
func (r *Registry) AllDefaultThemes() []model.Theme {
	var all []model.Theme
	for _, m := range r.modes {
		all = append(all, m.DefaultThemes()...)
	}
	return all
}

// FieldGroup is a named group of config fields for UI rendering.
type FieldGroup struct {
	Name   string
	Fields []ConfigField
}

// GroupedFields groups config fields by their Group label.
func GroupedFields(fields []ConfigField) []FieldGroup {
	groupMap := make(map[string][]ConfigField)
	var groupOrder []string
	for _, f := range fields {
		g := f.Group
		if g == "" {
			g = "Settings"
		}
		if _, seen := groupMap[g]; !seen {
			groupOrder = append(groupOrder, g)
		}
		groupMap[g] = append(groupMap[g], f)
	}
	groups := make([]FieldGroup, len(groupOrder))
	for i, name := range groupOrder {
		groups[i] = FieldGroup{Name: name, Fields: groupMap[name]}
	}
	return groups
}

// SortedConfigKeys returns config field keys sorted alphabetically.
func SortedConfigKeys(fields []ConfigField) []string {
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = f.Key
	}
	sort.Strings(keys)
	return keys
}
