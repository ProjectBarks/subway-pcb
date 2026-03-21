package community

import (
	"encoding/json"

	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

// defaultConfigJSON extracts a {key: default} map from a plugin's ConfigFields JSON.
// Returns "{}" when the raw bytes are empty or unparseable.
func defaultConfigJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var fields []plugin.ConfigField
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "{}"
	}
	b, _ := json.Marshal(plugin.DefaultConfigMap(fields))
	return string(b)
}
