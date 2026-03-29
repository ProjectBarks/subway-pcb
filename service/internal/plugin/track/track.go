package track

import (
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/lua"
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

// MTA route keys in display order.
var routeKeys = []string{
	"1", "2", "3",
	"4", "5", "6",
	"7",
	"A", "C", "E",
	"B", "D", "F", "M",
	"G",
	"J", "Z",
	"L",
	"N", "Q", "R", "W",
	"S", "FS", "GS",
	"SI",
}

// Plugin renders the standard subway map view.
type Plugin struct{}

func (p *Plugin) Name() string               { return "track" }
func (p *Plugin) Description() string        { return "Live subway map — trains at their current stations" }
func (p *Plugin) RequiredFeatures() []string  { return []string{"mta-stations"} }
func (p *Plugin) LuaSource() string           { return lua.TrackSource }

func (p *Plugin) ConfigFields() []plugin.ConfigField {
	fields := make([]plugin.ConfigField, 0, len(routeKeys)+1)
	fields = append(fields, plugin.ConfigField{
		Key: "brightness", Label: "Brightness", Type: plugin.FieldNumber,
		Default: "255", Min: "1", Max: "255", Group: "Settings",
	})
	for _, key := range routeKeys {
		fields = append(fields, plugin.ConfigField{
			Key:     key,
			Label:   key,
			Type:    plugin.FieldColor,
			Default: classicMTA[key],
			Group:   "Route Colors",
		})
	}
	return fields
}

func (p *Plugin) DefaultPresets() []model.Preset {
	now := time.Now()
	preset := func(id, name string, vals map[string]string) model.Preset {
		return model.Preset{ID: id, Name: name, PluginName: "track", IsBuiltIn: true, Values: vals, CreatedAt: now, UpdatedAt: now}
	}
	return []model.Preset{
		preset("track-classic-mta", "Classic MTA", classicMTA),
		preset("track-night", "Night Mode", nightMode),
		preset("track-sunset", "Sunset", sunset),
		preset("track-ocean", "Ocean", ocean),
		preset("track-monochrome", "Monochrome", monochrome),
	}
}
