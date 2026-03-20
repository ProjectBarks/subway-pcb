package dashboard

import "github.com/ProjectBarks/subway-pcb/service/internal/model"

// BoardCard pairs a device with its active preset for dashboard display.
type BoardCard struct {
	Device           model.Device
	Preset           *model.Preset
	ActivePluginName string // resolved human-readable plugin name
}
