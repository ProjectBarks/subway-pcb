package store

import "github.com/ProjectBarks/subway-pcb/server/internal/model"

// Store defines the persistence interface for all domain objects.
type Store interface {
	// Devices
	GetDevice(mac string) (*model.Device, error)
	ListDevices() ([]model.Device, error)
	ListDevicesByUser(email string) ([]model.Device, error)
	UpsertDevice(d *model.Device) error
	UpdateDeviceLastSeen(mac string) error

	// Access
	GrantAccess(a *model.DeviceAccess) error
	RevokeAccess(mac, email string) error
	ListAccessByDevice(mac string) ([]model.DeviceAccess, error)
	ListAccessByUser(email string) ([]model.DeviceAccess, error)
	HasAccess(mac, email string) (bool, error)

	// Presets
	GetPreset(id string) (*model.Preset, error)
	ListPresets() ([]model.Preset, error)
	ListPresetsByOwner(email string) ([]model.Preset, error)
	CreatePreset(t *model.Preset) error
	UpdatePreset(t *model.Preset) error
	DeletePreset(id string) error

	// Users
	GetUser(email string) (*model.User, error)
	UpsertUser(u *model.User) error
	ListUsers() ([]model.User, error)

	// Plugins
	GetPlugin(id string) (*model.Plugin, error)
	ListPlugins() ([]model.Plugin, error)
	ListPublishedPlugins() ([]model.Plugin, error)
	ListPluginsByAuthor(email string) ([]model.Plugin, error)
	SearchPlugins(query, sort string) ([]model.Plugin, error)
	CreatePlugin(p *model.Plugin) error
	UpdatePlugin(p *model.Plugin) error
	DeletePlugin(id string) error
	IncrementPluginInstalls(id string) error

	// User Plugins (installed)
	InstallPlugin(userEmail, pluginID string) error
	UninstallPlugin(userEmail, pluginID string) error
	ListInstalledPlugins(userEmail string) ([]model.Plugin, error)
	IsPluginInstalled(userEmail, pluginID string) (bool, error)

	// Lifecycle
	Close() error
}
