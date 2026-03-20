package track

import (
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/plugin"
)

// MTA route keys in display order.
var routeKeys = []string{
	"ROUTE_1", "ROUTE_2", "ROUTE_3",
	"ROUTE_4", "ROUTE_5", "ROUTE_6",
	"ROUTE_7",
	"ROUTE_A", "ROUTE_C", "ROUTE_E",
	"ROUTE_B", "ROUTE_D", "ROUTE_F", "ROUTE_M",
	"ROUTE_G",
	"ROUTE_J", "ROUTE_Z",
	"ROUTE_L",
	"ROUTE_N", "ROUTE_Q", "ROUTE_R", "ROUTE_W",
	"ROUTE_S", "ROUTE_FS", "ROUTE_GS",
	"ROUTE_SI",
}

// Plugin renders the standard subway map view.
type Plugin struct{}

func (p *Plugin) Name() string        { return "track" }
func (p *Plugin) Description() string { return "Live subway map — trains at their current stations" }

func (p *Plugin) ConfigFields() []plugin.ConfigField {
	fields := make([]plugin.ConfigField, 0, len(routeKeys))
	for _, key := range routeKeys {
		fields = append(fields, plugin.ConfigField{
			Key:     key,
			Label:   key[6:], // "ROUTE_1" -> "1"
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

func (p *Plugin) Render(ctx plugin.RenderContext) ([]byte, error) {
	trains := ctx.Aggregator.GetStationTrains()
	fields := p.ConfigFields()
	pixels := make([]byte, ctx.TotalLEDs*3)

	for i := 0; i < ctx.TotalLEDs; i++ {
		if i >= len(ctx.StationIDs) {
			break
		}
		sid := ctx.StationIDs[i]
		if sid == "" {
			continue
		}
		info, active := trains[sid]
		if !active {
			continue
		}
		r, g, b := ctx.ConfigColor(info.Route.String(), fields)
		pixels[i*3+0] = r
		pixels[i*3+1] = g
		pixels[i*3+2] = b
	}
	return pixels, nil
}
