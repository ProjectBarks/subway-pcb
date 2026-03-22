package board

import (
	"encoding/json"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

func deviceTitle(d *model.Device) string {
	if d.Name != "" {
		return d.Name
	}
	return d.MAC
}

// configJSON encodes a config map as a JSON string for embedding in data attributes.
func configJSON(config map[string]string) string {
	b, _ := json.Marshal(config)
	return string(b)
}

// dbPluginDefaultConfigJSON returns JSON of a DB plugin's default config values.
func dbPluginDefaultConfigJSON(p model.Plugin) string {
	if len(p.ConfigFields) == 0 {
		return "{}"
	}
	var fields []plugin.ConfigField
	if err := json.Unmarshal(p.ConfigFields, &fields); err != nil {
		return "{}"
	}
	b, _ := json.Marshal(plugin.DefaultConfigMap(fields))
	return string(b)
}
