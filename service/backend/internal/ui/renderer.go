package ui

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// Renderer manages HTML template rendering.
// Each page template gets its own clone of the base (layouts + partials) so
// that multiple pages can define {{define "content"}} without colliding.
type Renderer struct {
	pages    map[string]*template.Template // page name -> compiled template set
	partials *template.Template            // shared partials-only set for rendering fragments
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"colorHex": func(c [3]uint8) string {
			return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
		},
		"routeName": func(key string) string {
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
}

// NewRenderer loads templates from the given directory.
// Layouts + partials form a "base" set. Each page template is cloned from the
// base so that {{define "content"}} in one page doesn't stomp another.
func NewRenderer(templateDir string) (*Renderer, error) {
	fm := funcMap()

	// 1. Parse layouts + partials into a shared base template
	base := template.New("base").Funcs(fm)

	for _, sub := range []string{"layouts", "partials"} {
		pattern := filepath.Join(templateDir, sub, "*.html")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", pattern, err)
		}
		for _, match := range matches {
			if _, err := base.ParseFiles(match); err != nil {
				return nil, fmt.Errorf("parse %s: %w", match, err)
			}
		}
	}

	// 2. For each page, clone the base and parse the page template into it
	pagePattern := filepath.Join(templateDir, "pages", "*.html")
	pageFiles, err := filepath.Glob(pagePattern)
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for _, pf := range pageFiles {
		// Derive page name from filename: "dashboard.html" -> "dashboard"
		name := strings.TrimSuffix(filepath.Base(pf), ".html")

		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone base for %s: %w", name, err)
		}

		if _, err := clone.ParseFiles(pf); err != nil {
			return nil, fmt.Errorf("parse page %s: %w", pf, err)
		}

		pages[name] = clone
		log.Printf("ui: registered page template %q", name)
	}

	return &Renderer{pages: pages, partials: base}, nil
}

// Render executes a page template. For full pages (dashboard, board, login_required)
// it executes "base" within that page's template set. For partials (board_controls,
// board_grid, device_access, etc.) it renders from the shared partials set.
func (r *Renderer) Render(w io.Writer, name string, data any) error {
	// Check if it's a page template
	if tmpl, ok := r.pages[name]; ok {
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("ui: render page %q error: %v", name, err)
			return err
		}
		return nil
	}

	// Otherwise render a partial by name
	if err := r.partials.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("ui: render partial %q error: %v", name, err)
		return err
	}
	return nil
}
