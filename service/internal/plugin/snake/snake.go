package snake

import (
	"fmt"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/lua"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

// Plugin renders per-strip snakes with independent colors.
type Plugin struct{}

func (p *Plugin) Name() string               { return "snake" }
func (p *Plugin) Description() string        { return "Animated snakes running across each LED strip" }
func (p *Plugin) RequiredFeatures() []string  { return nil }
func (p *Plugin) LuaSource() string           { return lua.SnakeSource }

func (p *Plugin) ConfigFields() []plugin.ConfigField {
	defaults := rainbow()
	fields := make([]plugin.ConfigField, 0, 12)
	for i := range 9 {
		key := fmt.Sprintf("strip_%d_color", i+1)
		fields = append(fields, plugin.ConfigField{
			Key: key, Label: fmt.Sprintf("Strip %d", i+1),
			Type: plugin.FieldColor, Default: defaults[key], Group: "Strip Colors",
		})
	}
	fields = append(fields,
		plugin.ConfigField{Key: "brightness", Label: "Brightness", Type: plugin.FieldNumber, Default: "255", Min: "1", Max: "255", Group: "Settings"},
		plugin.ConfigField{Key: "snake_length", Label: "Snake Length", Type: plugin.FieldNumber, Default: "5", Min: "1", Max: "30", Group: "Settings"},
		plugin.ConfigField{Key: "snake_count", Label: "Number of Snakes", Type: plugin.FieldNumber, Default: "1", Min: "1", Max: "5", Group: "Settings"},
		plugin.ConfigField{Key: "speed_ms", Label: "Step Delay (ms)", Type: plugin.FieldNumber, Default: "2000", Min: "50", Max: "5000", Group: "Settings"},
	)
	return fields
}

func (p *Plugin) DefaultPresets() []model.Preset {
	now := time.Now()
	preset := func(id, name string, vals map[string]string) model.Preset {
		return model.Preset{ID: id, Name: name, PluginName: "snake", IsBuiltIn: true, Values: vals, CreatedAt: now, UpdatedAt: now}
	}
	return []model.Preset{
		preset("snake-rainbow", "Rainbow", rainbow()),
		preset("snake-fire", "Fire", fire()),
		preset("snake-ice", "Ice", ice()),
		preset("snake-neon", "Neon", neon()),
		preset("snake-mono", "Monochrome", mono()),
	}
}
