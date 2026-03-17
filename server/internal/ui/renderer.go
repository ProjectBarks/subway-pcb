package ui

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"time"
)

// Renderer manages HTML template rendering.
type Renderer struct {
	templates *template.Template
}

// NewRenderer loads and parses all templates from the given directory.
func NewRenderer(templateDir string) (*Renderer, error) {
	funcMap := template.FuncMap{
		"colorHex": func(c [3]uint8) string {
			return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
		},
		"routeName": func(key string) string {
			// "ROUTE_1" -> "1", "ROUTE_SI" -> "SI"
			if len(key) > 6 {
				return key[6:]
			}
			return key
		},
		"timeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "never"
			}
			d := time.Since(t)
			switch {
			case d < time.Minute:
				return fmt.Sprintf("%ds ago", int(d.Seconds()))
			case d < time.Hour:
				return fmt.Sprintf("%dm ago", int(d.Minutes()))
			case d < 24*time.Hour:
				return fmt.Sprintf("%dh ago", int(d.Hours()))
			default:
				return fmt.Sprintf("%dd ago", int(d.Hours()/24))
			}
		},
		"isAdmin": func(role string) bool {
			return role == "admin"
		},
		"isOnline": func(lastSeen time.Time) bool {
			return time.Since(lastSeen) < 30*time.Second
		},
		"seq": func(start, end int) []int {
			s := make([]int, 0, end-start)
			for i := start; i < end; i++ {
				s = append(s, i)
			}
			return s
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				m[key] = values[i+1]
			}
			return m
		},
	}

	patterns := []string{
		filepath.Join(templateDir, "layouts", "*.html"),
		filepath.Join(templateDir, "partials", "*.html"),
		filepath.Join(templateDir, "pages", "*.html"),
	}

	tmpl := template.New("").Funcs(funcMap)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob templates %s: %w", pattern, err)
		}
		for _, match := range matches {
			if _, err := tmpl.ParseFiles(match); err != nil {
				return nil, fmt.Errorf("parse template %s: %w", match, err)
			}
		}
	}

	return &Renderer{templates: tmpl}, nil
}

// Render executes a named template with the given data.
func (r *Renderer) Render(w io.Writer, name string, data any) error {
	return r.templates.ExecuteTemplate(w, name, data)
}
