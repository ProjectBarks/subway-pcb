package dashboard

import "github.com/ProjectBarks/subway-pcb/service/internal/model"

// BoardCard pairs a device with its active preset for dashboard display.
type BoardCard struct {
	Device           model.Device
	Preset           *model.Preset
	ActivePluginName string // resolved human-readable plugin name
	BoardModelName   string // e.g. "NYC Subway", from board manifest
	LuaSource        string // resolved Lua source for active plugin
	ConfigJSON       string // JSON-encoded merged config map[string]string
	BoardURL         string // e.g. "/static/dist/boards/nyc-subway/v1/board.json"
}
