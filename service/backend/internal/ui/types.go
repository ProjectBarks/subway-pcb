package ui

import (
	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
)

// BoardCard pairs a device with its active theme for dashboard display.
type BoardCard struct {
	Device model.Device
	Theme  *model.Theme
}

// BoardData holds all data needed to render the board detail page and its partials.
type BoardData struct {
	User         *model.User
	Device       *model.Device
	Themes       []model.Theme
	Access       []model.DeviceAccess
	Modes        []mode.Mode
	Boards       []model.Device
	ActiveMAC    string
	ConfigGroups []mode.FieldGroup
	ConfigValues map[string]string
}
