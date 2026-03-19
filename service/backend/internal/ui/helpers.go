package ui

import (
	"fmt"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

// TimeAgo formats a time as a human-readable relative string.
func TimeAgo(t time.Time) string {
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
}

// IsOnline returns true if the device was seen within 30 seconds.
func IsOnline(lastSeen time.Time) bool {
	return time.Since(lastSeen) < 30*time.Second
}

func lenStr(boards []model.Device) string {
	return fmt.Sprintf("%d", len(boards))
}

func plural(boards []model.Device) string {
	if len(boards) != 1 {
		return "s"
	}
	return ""
}

func deviceTitle(d *model.Device) string {
	if d.Name != "" {
		return d.Name
	}
	return d.MAC
}
