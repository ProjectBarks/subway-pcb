package plugin

// FieldType controls how the UI renders a config field.
type FieldType string

const (
	FieldColor  FieldType = "color"
	FieldNumber FieldType = "number"
	FieldSelect FieldType = "select"
)

// ConfigField describes a single configurable parameter for a mode.
type ConfigField struct {
	Key     string    `json:"key"`
	Label   string    `json:"label"`
	Type    FieldType `json:"type"`
	Default string    `json:"default"`
	Group   string    `json:"group,omitempty"`
	Min     string    `json:"min,omitempty"`
	Max     string    `json:"max,omitempty"`
	Options []string  `json:"options,omitempty"`
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
