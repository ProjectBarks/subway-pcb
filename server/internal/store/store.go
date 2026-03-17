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

	// Themes 
	GetTheme(id string) (*model.Theme, error)
	ListThemes() ([]model.Theme, error)
	ListThemesByOwner(email string) ([]model.Theme, error)
	CreateTheme(t *model.Theme) error
	UpdateTheme(t *model.Theme) error
	DeleteTheme(id string) error

	// Users
	GetUser(email string) (*model.User, error)
	UpsertUser(u *model.User) error
	ListUsers() ([]model.User, error)

	// Lifecycle
	Close() error
}
