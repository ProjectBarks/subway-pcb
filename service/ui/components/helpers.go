package components

import (
	"fmt"
	"time"
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
	return time.Since(lastSeen) < 30 * time.Second
}
