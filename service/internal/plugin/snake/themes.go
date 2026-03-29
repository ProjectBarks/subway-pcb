package snake

import "fmt"

// theme builds a Values map from 9 strip colors and default settings.
func theme(strips [9]string) map[string]string {
	m := map[string]string{"brightness": "255", "snake_length": "5", "snake_count": "1", "speed_ms": "2000"}
	for i, c := range strips {
		m[fmt.Sprintf("strip_%d_color", i+1)] = c
	}
	return m
}

// uniform returns 9 copies of the same color.
func uniform(hex string) [9]string {
	return [9]string{hex, hex, hex, hex, hex, hex, hex, hex, hex}
}

// gradient interpolates 9 colors between two RGB endpoints.
func gradient(r1, g1, b1, r2, g2, b2 uint8) [9]string {
	var out [9]string
	for i := range 9 {
		t := float64(i) / 8.0
		r := uint8(float64(r1)*(1-t) + float64(r2)*t)
		g := uint8(float64(g1)*(1-t) + float64(g2)*t)
		b := uint8(float64(b1)*(1-t) + float64(b2)*t)
		out[i] = fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return out
}

func rainbow() map[string]string {
	return theme([9]string{
		"#ff0000", "#00ff00", "#0000ff",
		"#ffff00", "#00ffff", "#ff00ff",
		"#ff8000", "#8000ff", "#00ff80",
	})
}

func fire() map[string]string { return theme(gradient(0xff, 0x00, 0x00, 0xff, 0xcc, 0x00)) }
func ice() map[string]string  { return theme(gradient(0x00, 0xbf, 0xff, 0xff, 0xff, 0xff)) }
func mono() map[string]string { return theme(uniform("#ffffff")) }

func neon() map[string]string {
	return theme([9]string{
		"#ff0080", "#00ff80", "#8000ff",
		"#ff8000", "#0080ff", "#80ff00",
		"#ff0040", "#00ffff", "#ff00ff",
	})
}
