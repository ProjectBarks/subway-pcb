package model

import (
	"encoding/json"
	"time"
)

// Device represents a physical subway PCB board.
type Device struct {
	MAC          string            `json:"mac"`
	Name         string            `json:"name"`
	BoardModelID string            `json:"board_model_id"` // e.g. "nyc-subway/v1"
	PluginName   string            `json:"plugin_name"`
	PresetID     string            `json:"theme_id"`
	FirmwareVer  string            `json:"firmware_ver"`
	PluginConfig map[string]string `json:"plugin_config,omitempty"`
	LastSeen     time.Time         `json:"last_seen"`
	CreatedAt    time.Time         `json:"created_at"`
}

// DeviceAccess grants a user access to a specific device.
type DeviceAccess struct {
	MAC       string    `json:"mac"`
	UserEmail string    `json:"user_email"`
	GrantedBy string    `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
}

// Preset is a named preset of key-value config for a specific plugin.
// Values map directly to the plugin's ConfigFields keys.
type Preset struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	PluginName string            `json:"plugin_name"`
	OwnerEmail string            `json:"owner_email"` // empty for built-in
	IsBuiltIn  bool              `json:"is_built_in"`
	Values     map[string]string `json:"values"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// User represents an authenticated user.
type User struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"` // "admin" or "user"
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}

// IsAdmin returns true if the user has the admin role.
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// Plugin represents a community or built-in plugin stored in the database.
type Plugin struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`         // "builtin" or "lua"
	AuthorEmail      string          `json:"author_email"`
	Description      string          `json:"description"`
	Category         string          `json:"category"` // ambient, data-driven, reactive, artistic
	LuaSource        string          `json:"lua_source"`
	ConfigFields     json.RawMessage `json:"config_fields,omitempty"`
	RequiredFeatures []string        `json:"required_features,omitempty"`
	Installs         int             `json:"installs"`
	IsPublished      bool            `json:"is_published"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// UserPlugin tracks which plugins a user has installed.
type UserPlugin struct {
	UserEmail   string    `json:"user_email"`
	PluginID    string    `json:"plugin_id"`
	InstalledAt time.Time `json:"installed_at"`
}
