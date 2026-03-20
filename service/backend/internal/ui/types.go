package ui

import (
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/plugin"
)

// BoardCard pairs a device with its active preset for dashboard display.
type BoardCard struct {
	Device model.Device
	Preset *model.Preset
}

// BoardData holds all data needed to render the board detail page and its partials.
type BoardData struct {
	User             *model.User
	Device           *model.Device
	Presets          []model.Preset
	Access           []model.DeviceAccess
	Plugins          []plugin.Plugin       // built-in renderers (track, snake)
	InstalledPlugins []model.Plugin        // user-installed DB plugins
	Boards           []model.Device
	ActiveMAC        string
	ConfigGroups     []plugin.FieldGroup
	ConfigValues     map[string]string
}

// ActivePluginName returns the human-readable name for the active plugin.
func (d BoardData) ActivePluginName() string {
	pn := d.Device.PluginName
	if pn == "" {
		return "track"
	}
	for _, p := range d.Plugins {
		if p.Name() == pn {
			return p.Name()
		}
	}
	for _, ip := range d.InstalledPlugins {
		if ip.ID == pn {
			return ip.Name
		}
	}
	// Fallback: plugin may have been uninstalled or deleted
	return "Unknown Plugin"
}
