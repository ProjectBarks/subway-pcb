package mode

import (
	"math"
	"time"
)

const (
	snakeLength = 15  // number of LEDs in the snake
	snakeSpeed  = 30  // LEDs per second
)

// SnakeMode renders a snake that continuously traverses all LEDs.
// Position is computed from time.Now() so it's stateless.
type SnakeMode struct{}

func (m *SnakeMode) Name() string        { return "snake" }
func (m *SnakeMode) Description() string { return "Animated snake traversing the LED strip" }

func (m *SnakeMode) Render(ctx RenderContext) ([]byte, error) {
	pixels := make([]byte, ctx.TotalLEDs*3)

	// Pick the snake color from the theme's first route color, or default to gold
	var snakeR, snakeG, snakeB uint8 = 201, 168, 48 // default gold #c9a830
	if ctx.Theme != nil {
		if c, ok := ctx.Theme.RouteColors["ROUTE_1"]; ok {
			snakeR, snakeG, snakeB = c[0], c[1], c[2]
		}
	}

	// Compute head position from wall clock — makes it stateless
	now := float64(time.Now().UnixMilli()) / 1000.0
	pathLen := ctx.TotalLEDs
	head := int(math.Mod(now*snakeSpeed, float64(pathLen*2)))

	// Bounce: if head >= pathLen, reverse direction
	if head >= pathLen {
		head = pathLen*2 - head - 1
	}

	for seg := 0; seg < snakeLength; seg++ {
		idx := head - seg
		if idx < 0 || idx >= pathLen {
			continue
		}

		// Fade tail: brightness decreases linearly
		fade := 1.0 - float64(seg)/float64(snakeLength)
		fade = fade * fade // quadratic falloff for smoother look

		pixels[idx*3+0] = uint8(float64(snakeR) * fade)
		pixels[idx*3+1] = uint8(float64(snakeG) * fade)
		pixels[idx*3+2] = uint8(float64(snakeB) * fade)
	}

	return pixels, nil
}
