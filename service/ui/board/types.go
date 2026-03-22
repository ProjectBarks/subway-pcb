package board

import (
	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/plugin"
)

// BoardData holds all data needed to render the board detail page and its partials.
type BoardData struct {
	User             *model.User
	Device           *model.Device
	Presets          []model.Preset
	Access           []model.DeviceAccess
	Plugins          []model.Plugin        // compatible plugins (built-in + published)
	InstalledPlugins []model.Plugin        // user-installed DB plugins
	Boards           []model.Device
	ActiveMAC        string
	ConfigGroups     []plugin.FieldGroup
	ConfigValues     map[string]string
	BoardURL         string                // URL path to board.json for 3D viewer
	ActiveLuaSource  string                // Lua source for the active plugin
}

// ActivePluginName returns the human-readable name for the active plugin.
func (d BoardData) ActivePluginName() string {
	pn := d.Device.PluginName
	if pn == "" {
		return ""
	}
	for _, p := range d.Plugins {
		if p.ID == pn {
			return p.Name
		}
	}
	for _, ip := range d.InstalledPlugins {
		if ip.ID == pn {
			return ip.Name
		}
	}
	return "Unknown Plugin"
}
