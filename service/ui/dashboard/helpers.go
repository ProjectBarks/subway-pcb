package dashboard

import (
	"fmt"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
)

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
