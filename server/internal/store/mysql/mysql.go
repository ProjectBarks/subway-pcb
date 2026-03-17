package mysql

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ---------- GORM model structs ----------

type Device struct {
	MAC         string         `gorm:"column:mac;primaryKey;size:17"`
	Name        string         `gorm:"column:name;size:255"`
	Mode        string         `gorm:"column:mode;size:50;default:'idle'"`
	ThemeID     string         `gorm:"column:theme_id;size:36"`
	FirmwareVer string         `gorm:"column:firmware_ver;size:32"`
	ModeConfig  datatypes.JSON `gorm:"column:mode_config;type:json"`
	LastSeen    time.Time      `gorm:"column:last_seen;autoUpdateTime"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
}

type DeviceAccess struct {
	MAC       string    `gorm:"column:mac;primaryKey;size:17"`
	UserEmail string    `gorm:"column:user_email;primaryKey;size:255"`
	GrantedBy string    `gorm:"column:granted_by;size:255"`
	GrantedAt time.Time `gorm:"column:granted_at;autoCreateTime"`
}

type Theme struct {
	ID         string         `gorm:"column:id;primaryKey;size:50"`
	Name       string         `gorm:"column:name;size:255"`
	ModeName   string         `gorm:"column:mode_name;size:50;index"`
	OwnerEmail string         `gorm:"column:owner_email;size:255;index"`
	IsBuiltIn  bool           `gorm:"column:is_built_in;default:false"`
	Values     datatypes.JSON `gorm:"column:vals;type:json"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

type User struct {
	Email     string    `gorm:"column:email;primaryKey;size:255"`
	Name      string    `gorm:"column:name;size:255"`
	Role      string    `gorm:"column:role;size:20;default:'user'"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	LastSeen  time.Time `gorm:"column:last_seen;autoUpdateTime"`
}

// ---------- MySQLStore ----------

// MySQLStore implements store.Store backed by MySQL via GORM.
type MySQLStore struct {
	db *gorm.DB
}

// New opens a MySQL connection using the provided DSN and auto-migrates all
// tables. The returned MySQLStore satisfies the store.Store interface.
func New(dsn string) (*MySQLStore, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Device{}, &DeviceAccess{}, &Theme{}, &User{}); err != nil {
		return nil, err
	}

	return &MySQLStore{db: db}, nil
}

// ---------- Devices ----------

func (s *MySQLStore) GetDevice(mac string) (*model.Device, error) {
	var d Device
	if err := s.db.First(&d, "mac = ?", mac).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := toModelDevice(&d)
	return &out, nil
}

func (s *MySQLStore) ListDevices() ([]model.Device, error) {
	var rows []Device
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelDevices(rows), nil
}

func (s *MySQLStore) ListDevicesByUser(email string) ([]model.Device, error) {
	var rows []Device
	if err := s.db.
		Where("mac IN (?)", s.db.Model(&DeviceAccess{}).Select("mac").Where("user_email = ?", email)).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelDevices(rows), nil
}

func (s *MySQLStore) UpsertDevice(d *model.Device) error {
	row := fromModelDevice(d)
	return s.db.Save(&row).Error
}

func (s *MySQLStore) UpdateDeviceLastSeen(mac string) error {
	return s.db.Model(&Device{}).Where("mac = ?", mac).Update("last_seen", time.Now()).Error
}

// ---------- Access ----------

func (s *MySQLStore) GrantAccess(a *model.DeviceAccess) error {
	row := fromModelDeviceAccess(a)
	return s.db.Create(&row).Error
}

func (s *MySQLStore) RevokeAccess(mac, email string) error {
	return s.db.Where("mac = ? AND user_email = ?", mac, email).Delete(&DeviceAccess{}).Error
}

func (s *MySQLStore) ListAccessByDevice(mac string) ([]model.DeviceAccess, error) {
	var rows []DeviceAccess
	if err := s.db.Where("mac = ?", mac).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelDeviceAccesses(rows), nil
}

func (s *MySQLStore) ListAccessByUser(email string) ([]model.DeviceAccess, error) {
	var rows []DeviceAccess
	if err := s.db.Where("user_email = ?", email).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelDeviceAccesses(rows), nil
}

func (s *MySQLStore) HasAccess(mac, email string) (bool, error) {
	var count int64
	if err := s.db.Model(&DeviceAccess{}).Where("mac = ? AND user_email = ?", mac, email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---------- Themes ----------

func (s *MySQLStore) GetTheme(id string) (*model.Theme, error) {
	var t Theme
	if err := s.db.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out, err := toModelTheme(&t)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *MySQLStore) ListThemes() ([]model.Theme, error) {
	var rows []Theme
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelThemes(rows)
}

func (s *MySQLStore) ListThemesByOwner(email string) ([]model.Theme, error) {
	var rows []Theme
	if err := s.db.Where("owner_email = ?", email).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelThemes(rows)
}

func (s *MySQLStore) CreateTheme(t *model.Theme) error {
	row, err := fromModelTheme(t)
	if err != nil {
		return err
	}
	return s.db.Create(&row).Error
}

func (s *MySQLStore) UpdateTheme(t *model.Theme) error {
	row, err := fromModelTheme(t)
	if err != nil {
		return err
	}
	return s.db.Save(&row).Error
}

func (s *MySQLStore) DeleteTheme(id string) error {
	return s.db.Where("id = ?", id).Delete(&Theme{}).Error
}

// ---------- Users ----------

func (s *MySQLStore) GetUser(email string) (*model.User, error) {
	var u User
	if err := s.db.First(&u, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := toModelUser(&u)
	return &out, nil
}

func (s *MySQLStore) UpsertUser(u *model.User) error {
	row := fromModelUser(u)
	return s.db.Save(&row).Error
}

func (s *MySQLStore) ListUsers() ([]model.User, error) {
	var rows []User
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelUsers(rows), nil
}

// ---------- Lifecycle ----------

func (s *MySQLStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ---------- Conversion helpers ----------

func toModelDevice(d *Device) model.Device {
	var mc map[string]string
	if len(d.ModeConfig) > 0 {
		_ = json.Unmarshal(d.ModeConfig, &mc)
	}
	return model.Device{
		MAC:         d.MAC,
		Name:        d.Name,
		Mode:        d.Mode,
		ThemeID:     d.ThemeID,
		FirmwareVer: d.FirmwareVer,
		ModeConfig:  mc,
		LastSeen:    d.LastSeen,
		CreatedAt:   d.CreatedAt,
	}
}

func toModelDevices(rows []Device) []model.Device {
	out := make([]model.Device, len(rows))
	for i := range rows {
		out[i] = toModelDevice(&rows[i])
	}
	return out
}

func fromModelDevice(d *model.Device) Device {
	var mc datatypes.JSON
	if d.ModeConfig != nil {
		mc, _ = json.Marshal(d.ModeConfig)
	}
	return Device{
		MAC:         d.MAC,
		Name:        d.Name,
		Mode:        d.Mode,
		ThemeID:     d.ThemeID,
		FirmwareVer: d.FirmwareVer,
		ModeConfig:  mc,
		LastSeen:    d.LastSeen,
		CreatedAt:   d.CreatedAt,
	}
}

func toModelDeviceAccess(a *DeviceAccess) model.DeviceAccess {
	return model.DeviceAccess{
		MAC:       a.MAC,
		UserEmail: a.UserEmail,
		GrantedBy: a.GrantedBy,
		GrantedAt: a.GrantedAt,
	}
}

func toModelDeviceAccesses(rows []DeviceAccess) []model.DeviceAccess {
	out := make([]model.DeviceAccess, len(rows))
	for i := range rows {
		out[i] = toModelDeviceAccess(&rows[i])
	}
	return out
}

func fromModelDeviceAccess(a *model.DeviceAccess) DeviceAccess {
	return DeviceAccess{
		MAC:       a.MAC,
		UserEmail: a.UserEmail,
		GrantedBy: a.GrantedBy,
		GrantedAt: a.GrantedAt,
	}
}

func toModelTheme(t *Theme) (model.Theme, error) {
	var vals map[string]string
	if len(t.Values) > 0 {
		if err := json.Unmarshal(t.Values, &vals); err != nil {
			return model.Theme{}, err
		}
	}
	return model.Theme{
		ID:         t.ID,
		Name:       t.Name,
		ModeName:   t.ModeName,
		OwnerEmail: t.OwnerEmail,
		IsBuiltIn:  t.IsBuiltIn,
		Values:     vals,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}, nil
}

func toModelThemes(rows []Theme) ([]model.Theme, error) {
	out := make([]model.Theme, len(rows))
	for i := range rows {
		t, err := toModelTheme(&rows[i])
		if err != nil {
			return nil, err
		}
		out[i] = t
	}
	return out, nil
}

func fromModelTheme(t *model.Theme) (Theme, error) {
	vals, err := json.Marshal(t.Values)
	if err != nil {
		return Theme{}, err
	}
	return Theme{
		ID:         t.ID,
		Name:       t.Name,
		ModeName:   t.ModeName,
		OwnerEmail: t.OwnerEmail,
		IsBuiltIn:  t.IsBuiltIn,
		Values:     datatypes.JSON(vals),
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}, nil
}

func toModelUser(u *User) model.User {
	return model.User{
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		LastSeen:  u.LastSeen,
	}
}

func toModelUsers(rows []User) []model.User {
	out := make([]model.User, len(rows))
	for i := range rows {
		out[i] = toModelUser(&rows[i])
	}
	return out
}

func fromModelUser(u *model.User) User {
	return User{
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		LastSeen:  u.LastSeen,
	}
}
