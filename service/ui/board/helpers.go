package board

import "github.com/ProjectBarks/subway-pcb/service/internal/model"

func deviceTitle(d *model.Device) string {
	if d.Name != "" {
		return d.Name
	}
	return d.MAC
}
