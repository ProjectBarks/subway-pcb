package mode

import (
	"fmt"
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
	Config     map[string]string // resolved config (field defaults -> theme -> device overrides)
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

// FieldGroup is a named group of config fields for UI rendering.
type FieldGroup struct {
	Name   string
	Fields []ConfigField
}

// GroupedFields groups config fields by their Group label, preserving order.
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
