package track

import (
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
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

// Mode renders the standard subway map view.
type Mode struct{}

func (m *Mode) Name() string        { return "track" }
func (m *Mode) Description() string { return "Live subway map — trains at their current stations" }

func (m *Mode) ConfigFields() []mode.ConfigField {
	fields := make([]mode.ConfigField, 0, len(routeKeys))
	for _, key := range routeKeys {
		fields = append(fields, mode.ConfigField{
			Key:     key,
			Label:   key[6:], // "ROUTE_1" -> "1"
			Type:    mode.FieldColor,
			Default: classicMTA[key],
			Group:   "Route Colors",
		})
	}
	return fields
}

func (m *Mode) DefaultThemes() []model.Theme {
	now := time.Now()
	theme := func(id, name string, vals map[string]string) model.Theme {
		return model.Theme{ID: id, Name: name, ModeName: "track", IsBuiltIn: true, Values: vals, CreatedAt: now, UpdatedAt: now}
	}
	return []model.Theme{
		theme("track-classic-mta", "Classic MTA", classicMTA),
		theme("track-night", "Night Mode", nightMode),
		theme("track-sunset", "Sunset", sunset),
		theme("track-ocean", "Ocean", ocean),
		theme("track-monochrome", "Monochrome", monochrome),
	}
}

func (m *Mode) Render(ctx mode.RenderContext) ([]byte, error) {
	trains := ctx.Aggregator.GetStationTrains()
	fields := m.ConfigFields()
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
