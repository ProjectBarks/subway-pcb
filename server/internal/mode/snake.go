package mode

import (
	"fmt"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

var stripSizes = [9]int{97, 102, 55, 81, 70, 21, 22, 19, 11}

// SnakeMode renders per-strip snakes with independent colors.
type SnakeMode struct{}

func (m *SnakeMode) Name() string        { return "snake" }
func (m *SnakeMode) Description() string { return "Animated snakes running across each LED strip" }

func (m *SnakeMode) ConfigFields() []ConfigField {
	fields := make([]ConfigField, 0, 12)
	defaults := snakeRainbow()
	for i := 0; i < 9; i++ {
		key := fmt.Sprintf("strip_%d_color", i+1)
		fields = append(fields, ConfigField{
			Key: key, Label: fmt.Sprintf("Strip %d", i+1),
			Type: FieldColor, Default: defaults[key], Group: "Strip Colors",
		})
	}
	fields = append(fields,
		ConfigField{Key: "snake_length", Label: "Snake Length", Type: FieldNumber, Default: "5", Min: "1", Max: "30", Group: "Settings"},
		ConfigField{Key: "snake_count", Label: "Number of Snakes", Type: FieldNumber, Default: "1", Min: "1", Max: "5", Group: "Settings"},
		ConfigField{Key: "speed_ms", Label: "Step Delay (ms)", Type: FieldNumber, Default: "250", Min: "50", Max: "1000", Group: "Settings"},
	)
	return fields
}

func (m *SnakeMode) DefaultThemes() []model.Theme {
	now := time.Now()
	return []model.Theme{
		{ID: "snake-rainbow", Name: "Rainbow", ModeName: "snake", IsBuiltIn: true, Values: snakeRainbow(), CreatedAt: now, UpdatedAt: now},
		{ID: "snake-fire", Name: "Fire", ModeName: "snake", IsBuiltIn: true, Values: snakeFire(), CreatedAt: now, UpdatedAt: now},
		{ID: "snake-ice", Name: "Ice", ModeName: "snake", IsBuiltIn: true, Values: snakeIce(), CreatedAt: now, UpdatedAt: now},
		{ID: "snake-neon", Name: "Neon", ModeName: "snake", IsBuiltIn: true, Values: snakeNeon(), CreatedAt: now, UpdatedAt: now},
		{ID: "snake-mono", Name: "Monochrome", ModeName: "snake", IsBuiltIn: true, Values: snakeMono(), CreatedAt: now, UpdatedAt: now},
	}
}

func (m *SnakeMode) Render(ctx RenderContext) ([]byte, error) {
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
		speedMs = 250
	}

	step := int(time.Now().UnixMilli()) / speedMs

	offset := 0
	for strip := 0; strip < 9; strip++ {
		sz := stripSizes[strip]
		key := fmt.Sprintf("strip_%d_color", strip+1)
		r, g, b := ctx.ConfigColor(key, fields)

		for sn := 0; sn < snakeCount; sn++ {
			snakeOffset := (sz * sn) / snakeCount
			startPixel := (step + snakeOffset) % sz
			for px := 0; px < snakeLength; px++ {
				idx := (startPixel + px) % sz
				flatIdx := offset + idx
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

// --- Snake theme presets ---

// snakeTheme builds a theme Values map from 9 hex colors and optional setting overrides.
func snakeTheme(strips [9]string, settings ...string) map[string]string {
	m := map[string]string{"snake_length": "5", "snake_count": "1", "speed_ms": "250"}
	for i, c := range strips {
		m[fmt.Sprintf("strip_%d_color", i+1)] = c
	}
	// Apply overrides: "snake_count", "3", "speed_ms", "100", ...
	for i := 0; i+1 < len(settings); i += 2 {
		m[settings[i]] = settings[i+1]
	}
	return m
}

// uniform returns 9 copies of the same color.
func uniform(hex string) [9]string {
	return [9]string{hex, hex, hex, hex, hex, hex, hex, hex, hex}
}

// gradient returns 9 colors interpolated from start to end.
func gradient(startR, startG, startB, endR, endG, endB uint8) [9]string {
	var out [9]string
	for i := range 9 {
		t := float64(i) / 8.0
		r := uint8(float64(startR)*(1-t) + float64(endR)*t)
		g := uint8(float64(startG)*(1-t) + float64(endG)*t)
		b := uint8(float64(startB)*(1-t) + float64(endB)*t)
		out[i] = fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return out
}

var snakePresets = map[string][9]string{
	"rainbow": {"#ff0000", "#00ff00", "#0000ff", "#ffff00", "#00ffff", "#ff00ff", "#ff8000", "#8000ff", "#00ff80"},
	"neon":    {"#ff0080", "#00ff80", "#8000ff", "#ff8000", "#0080ff", "#80ff00", "#ff0040", "#00ffff", "#ff00ff"},
}

func snakeRainbow() map[string]string { return snakeTheme(snakePresets["rainbow"]) }
func snakeFire() map[string]string    { return snakeTheme(gradient(0xff, 0x00, 0x00, 0xff, 0xcc, 0x00)) }
func snakeIce() map[string]string     { return snakeTheme(gradient(0x00, 0xbf, 0xff, 0xff, 0xff, 0xff)) }
func snakeNeon() map[string]string    { return snakeTheme(snakePresets["neon"]) }
func snakeMono() map[string]string    { return snakeTheme(uniform("#ffffff")) }
