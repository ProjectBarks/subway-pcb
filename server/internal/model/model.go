package model

import "time"

// Device represents a physical subway PCB board.
type Device struct {
	MAC         string            `json:"mac"`
	Name        string            `json:"name"`
	Mode        string            `json:"mode"`
	ThemeID     string            `json:"theme_id"`
	FirmwareVer string            `json:"firmware_ver"`
	ModeConfig  map[string]string `json:"mode_config,omitempty"`
	LastSeen    time.Time         `json:"last_seen"`
	CreatedAt   time.Time         `json:"created_at"`
}

// DeviceAccess grants a user access to a specific device.
type DeviceAccess struct {
	MAC       string    `json:"mac"`
	UserEmail string    `json:"user_email"`
	GrantedBy string    `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
}

// Theme is a named preset of key-value config for a specific mode.
// Values map directly to the mode's ConfigFields keys.
type Theme struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	ModeName   string            `json:"mode_name"`
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
