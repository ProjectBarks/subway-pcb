package mode

import (
	"fmt"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

// All MTA route keys in display order.
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

// TrackMode renders the standard subway map view.
type TrackMode struct{}

func (m *TrackMode) Name() string        { return "track" }
func (m *TrackMode) Description() string { return "Live subway map — trains at their current stations" }

func (m *TrackMode) ConfigFields() []ConfigField {
	fields := make([]ConfigField, 0, len(routeKeys))
	for _, key := range routeKeys {
		label := key[6:] // "ROUTE_1" -> "1"
		fields = append(fields, ConfigField{
			Key:     key,
			Label:   label,
			Type:    FieldColor,
			Default: classicMTA[key],
			Group:   "Route Colors",
		})
	}
	return fields
}

func (m *TrackMode) DefaultThemes() []model.Theme {
	now := time.Now()
	return []model.Theme{
		{ID: "track-classic-mta", Name: "Classic MTA", ModeName: "track", IsBuiltIn: true, Values: classicMTA, CreatedAt: now, UpdatedAt: now},
		{ID: "track-night", Name: "Night Mode", ModeName: "track", IsBuiltIn: true, Values: nightMode, CreatedAt: now, UpdatedAt: now},
		{ID: "track-sunset", Name: "Sunset", ModeName: "track", IsBuiltIn: true, Values: sunset, CreatedAt: now, UpdatedAt: now},
		{ID: "track-ocean", Name: "Ocean", ModeName: "track", IsBuiltIn: true, Values: ocean, CreatedAt: now, UpdatedAt: now},
		{ID: "track-monochrome", Name: "Monochrome", ModeName: "track", IsBuiltIn: true, Values: monochrome, CreatedAt: now, UpdatedAt: now},
	}
}

func (m *TrackMode) Render(ctx RenderContext) ([]byte, error) {
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
		routeKey := info.Route.String()
		r, g, b := ctx.ConfigColor(routeKey, fields)
		pixels[i*3+0] = r
		pixels[i*3+1] = g
		pixels[i*3+2] = b
	}
	return pixels, nil
}

// --- Built-in track themes ---

func hexRGB(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

var classicMTA = map[string]string{
	"ROUTE_1": "#ff0000", "ROUTE_2": "#ff0000", "ROUTE_3": "#ff0000",
	"ROUTE_4": "#00933c", "ROUTE_5": "#00933c", "ROUTE_6": "#00933c",
	"ROUTE_7": "#b933ad",
	"ROUTE_A": "#0039a6", "ROUTE_C": "#0039a6", "ROUTE_E": "#0039a6",
	"ROUTE_B": "#ff6319", "ROUTE_D": "#ff6319", "ROUTE_F": "#ff6319", "ROUTE_M": "#ff6319",
	"ROUTE_G":  "#6cbe45",
	"ROUTE_J":  "#996633", "ROUTE_Z": "#996633",
	"ROUTE_L":  "#a7a9ac",
	"ROUTE_N":  "#fccc0a", "ROUTE_Q": "#fccc0a", "ROUTE_R": "#fccc0a", "ROUTE_W": "#fccc0a",
	"ROUTE_S":  "#808183", "ROUTE_FS": "#808183", "ROUTE_GS": "#808183",
	"ROUTE_SI": "#0039a6",
}

var nightMode = map[string]string{
	"ROUTE_1": "#991111", "ROUTE_2": "#991111", "ROUTE_3": "#991111",
	"ROUTE_4": "#0a5c2a", "ROUTE_5": "#0a5c2a", "ROUTE_6": "#0a5c2a",
	"ROUTE_7": "#6b1f6b",
	"ROUTE_A": "#0a1f5c", "ROUTE_C": "#0a1f5c", "ROUTE_E": "#0a1f5c",
	"ROUTE_B": "#8c3a10", "ROUTE_D": "#8c3a10", "ROUTE_F": "#8c3a10", "ROUTE_M": "#8c3a10",
	"ROUTE_G":  "#3a6622",
	"ROUTE_J":  "#5c3a1a", "ROUTE_Z": "#5c3a1a",
	"ROUTE_L":  "#555555",
	"ROUTE_N":  "#8c7010", "ROUTE_Q": "#8c7010", "ROUTE_R": "#8c7010", "ROUTE_W": "#8c7010",
	"ROUTE_S":  "#444444", "ROUTE_FS": "#444444", "ROUTE_GS": "#444444",
	"ROUTE_SI": "#0a1f5c",
}

var sunset = map[string]string{
	"ROUTE_1": "#ff4500", "ROUTE_2": "#ff4500", "ROUTE_3": "#ff4500",
	"ROUTE_4": "#ff6347", "ROUTE_5": "#ff6347", "ROUTE_6": "#ff6347",
	"ROUTE_7": "#ff1493",
	"ROUTE_A": "#dc143c", "ROUTE_C": "#dc143c", "ROUTE_E": "#dc143c",
	"ROUTE_B": "#ff8c00", "ROUTE_D": "#ff8c00", "ROUTE_F": "#ff8c00", "ROUTE_M": "#ff8c00",
	"ROUTE_G":  "#ffa500",
	"ROUTE_J":  "#b22222", "ROUTE_Z": "#b22222",
	"ROUTE_L":  "#cd853f",
	"ROUTE_N":  "#ffd700", "ROUTE_Q": "#ffd700", "ROUTE_R": "#ffd700", "ROUTE_W": "#ffd700",
	"ROUTE_S":  "#8b4513", "ROUTE_FS": "#8b4513", "ROUTE_GS": "#8b4513",
	"ROUTE_SI": "#c71585",
}

var ocean = map[string]string{
	"ROUTE_1": "#00bfff", "ROUTE_2": "#00bfff", "ROUTE_3": "#00bfff",
	"ROUTE_4": "#20b2aa", "ROUTE_5": "#20b2aa", "ROUTE_6": "#20b2aa",
	"ROUTE_7": "#7b68ee",
	"ROUTE_A": "#4169e1", "ROUTE_C": "#4169e1", "ROUTE_E": "#4169e1",
	"ROUTE_B": "#00ced1", "ROUTE_D": "#00ced1", "ROUTE_F": "#00ced1", "ROUTE_M": "#00ced1",
	"ROUTE_G":  "#48d1cc",
	"ROUTE_J":  "#5f9ea0", "ROUTE_Z": "#5f9ea0",
	"ROUTE_L":  "#b0c4de",
	"ROUTE_N":  "#87ceeb", "ROUTE_Q": "#87ceeb", "ROUTE_R": "#87ceeb", "ROUTE_W": "#87ceeb",
	"ROUTE_S":  "#708090", "ROUTE_FS": "#708090", "ROUTE_GS": "#708090",
	"ROUTE_SI": "#1e90ff",
}

var monochrome = map[string]string{
	"ROUTE_1": "#ffffff", "ROUTE_2": "#ffffff", "ROUTE_3": "#ffffff",
	"ROUTE_4": "#ffffff", "ROUTE_5": "#ffffff", "ROUTE_6": "#ffffff",
	"ROUTE_7": "#ffffff",
	"ROUTE_A": "#ffffff", "ROUTE_C": "#ffffff", "ROUTE_E": "#ffffff",
	"ROUTE_B": "#ffffff", "ROUTE_D": "#ffffff", "ROUTE_F": "#ffffff", "ROUTE_M": "#ffffff",
	"ROUTE_G":  "#ffffff",
	"ROUTE_J":  "#ffffff", "ROUTE_Z": "#ffffff",
	"ROUTE_L":  "#ffffff",
	"ROUTE_N":  "#ffffff", "ROUTE_Q": "#ffffff", "ROUTE_R": "#ffffff", "ROUTE_W": "#ffffff",
	"ROUTE_S":  "#ffffff", "ROUTE_FS": "#ffffff", "ROUTE_GS": "#ffffff",
	"ROUTE_SI": "#ffffff",
}
