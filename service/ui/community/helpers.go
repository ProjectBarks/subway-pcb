package community

import "encoding/json"

// defaultConfigJSON extracts a {key: default} map from a plugin's ConfigFields JSON.
// Returns "{}" when the raw bytes are empty or unparseable.
func defaultConfigJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var fields []struct {
		Key     string `json:"key"`
		Default string `json:"default"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "{}"
	}
	config := make(map[string]string)
	for _, f := range fields {
		config[f.Key] = f.Default
	}
	b, _ := json.Marshal(config)
	return string(b)
}
