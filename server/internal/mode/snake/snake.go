package snake

import (
	"fmt"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

var stripSizes = [9]int{97, 102, 55, 81, 70, 21, 22, 19, 11}

// Mode renders per-strip snakes with independent colors.
type Mode struct{}

func (m *Mode) Name() string        { return "snake" }
func (m *Mode) Description() string { return "Animated snakes running across each LED strip" }

func (m *Mode) ConfigFields() []mode.ConfigField {
	defaults := rainbow()
	fields := make([]mode.ConfigField, 0, 12)
	for i := range 9 {
		key := fmt.Sprintf("strip_%d_color", i+1)
		fields = append(fields, mode.ConfigField{
			Key: key, Label: fmt.Sprintf("Strip %d", i+1),
			Type: mode.FieldColor, Default: defaults[key], Group: "Strip Colors",
		})
	}
	fields = append(fields,
		mode.ConfigField{Key: "snake_length", Label: "Snake Length", Type: mode.FieldNumber, Default: "5", Min: "1", Max: "30", Group: "Settings"},
		mode.ConfigField{Key: "snake_count", Label: "Number of Snakes", Type: mode.FieldNumber, Default: "1", Min: "1", Max: "5", Group: "Settings"},
		mode.ConfigField{Key: "speed_ms", Label: "Step Delay (ms)", Type: mode.FieldNumber, Default: "2000", Min: "50", Max: "5000", Group: "Settings"},
	)
	return fields
}

func (m *Mode) DefaultThemes() []model.Theme {
	now := time.Now()
	theme := func(id, name string, vals map[string]string) model.Theme {
		return model.Theme{ID: id, Name: name, ModeName: "snake", IsBuiltIn: true, Values: vals, CreatedAt: now, UpdatedAt: now}
	}
	return []model.Theme{
		theme("snake-rainbow", "Rainbow", rainbow()),
		theme("snake-fire", "Fire", fire()),
		theme("snake-ice", "Ice", ice()),
		theme("snake-neon", "Neon", neon()),
		theme("snake-mono", "Monochrome", mono()),
	}
}

func (m *Mode) Render(ctx mode.RenderContext) ([]byte, error) {
	fields := m.ConfigFields()
	pixels := make([]byte, ctx.TotalLEDs*3)

	snakeLength := ctx.ConfigInt("snake_length", fields)
	if snakeLength < 1 {
		snakeLength = 5
	}
	snakeCount := ctx.ConfigInt("snake_count", fields)
	if snakeCount < 1 {
		snakeCount = 1
	}
	speedMs := ctx.ConfigInt("speed_ms", fields)
	if speedMs < 50 {
		speedMs = 2000
	}

	step := int(time.Now().UnixMilli()) / speedMs

	offset := 0
	for strip := range 9 {
		sz := stripSizes[strip]
		r, g, b := ctx.ConfigColor(fmt.Sprintf("strip_%d_color", strip+1), fields)

		for sn := range snakeCount {
			snakeOffset := (sz * sn) / snakeCount
			startPixel := (step + snakeOffset) % sz
			for px := range snakeLength {
				flatIdx := offset + (startPixel+px)%sz
				if flatIdx < ctx.TotalLEDs {
					pixels[flatIdx*3+0] = r
					pixels[flatIdx*3+1] = g
					pixels[flatIdx*3+2] = b
				}
			}
		}
		offset += sz
	}
	return pixels, nil
}
