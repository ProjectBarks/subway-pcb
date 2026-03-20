package mysql

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ---------- GORM model structs ----------

type Device struct {
	MAC          string         `gorm:"column:mac;primaryKey;size:17"`
	Name         string         `gorm:"column:name;size:255"`
	BoardModelID string         `gorm:"column:board_model_id;size:50;default:'nyc-subway/v1'"`
	PluginName   string         `gorm:"column:plugin_name;size:50;default:'track'"`
	PresetID     string         `gorm:"column:theme_id;size:36"`
	FirmwareVer  string         `gorm:"column:firmware_ver;size:32"`
	PluginConfig datatypes.JSON `gorm:"column:plugin_config;type:json"`
	LastSeen     time.Time      `gorm:"column:last_seen;autoUpdateTime"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"`
}

type DeviceAccess struct {
	MAC       string    `gorm:"column:mac;primaryKey;size:17"`
	UserEmail string    `gorm:"column:user_email;primaryKey;size:255"`
	GrantedBy string    `gorm:"column:granted_by;size:255"`
	GrantedAt time.Time `gorm:"column:granted_at;autoCreateTime"`
}

type Preset struct {
	ID         string         `gorm:"column:id;primaryKey;size:50"`
	Name       string         `gorm:"column:name;size:255"`
	PluginName string         `gorm:"column:plugin_name;size:50;index"`
	OwnerEmail string         `gorm:"column:owner_email;size:255;index"`
	IsBuiltIn  bool           `gorm:"column:is_built_in;default:false"`
	Values     datatypes.JSON `gorm:"column:vals;type:json"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Preset) TableName() string { return "themes" }

type User struct {
	Email     string    `gorm:"column:email;primaryKey;size:255"`
	Name      string    `gorm:"column:name;size:255"`
	Role      string    `gorm:"column:role;size:20;default:'user'"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	LastSeen  time.Time `gorm:"column:last_seen;autoUpdateTime"`
}

type Plugin struct {
	ID           string         `gorm:"column:id;primaryKey;size:50"`
	Name         string         `gorm:"column:name;size:255"`
	Type         string         `gorm:"column:type;size:20;default:'lua'"`
	AuthorEmail  string         `gorm:"column:author_email;size:255;index"`
	Description  string         `gorm:"column:description;type:text"`
	Category     string         `gorm:"column:category;size:50"`
	LuaSource    string         `gorm:"column:lua_source;type:longtext"`
	ConfigFields datatypes.JSON `gorm:"column:config_fields;type:json"`
	Installs     int            `gorm:"column:installs;default:0"`
	IsPublished  bool           `gorm:"column:is_published;default:false"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

type UserPlugin struct {
	UserEmail   string    `gorm:"column:user_email;primaryKey;size:255"`
	PluginID    string    `gorm:"column:plugin_id;primaryKey;size:50"`
	InstalledAt time.Time `gorm:"column:installed_at;autoCreateTime"`
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

	if err := db.AutoMigrate(&Device{}, &DeviceAccess{}, &Preset{}, &User{}, &Plugin{}, &UserPlugin{}); err != nil {
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

// ---------- Presets ----------

func (s *MySQLStore) GetPreset(id string) (*model.Preset, error) {
	var t Preset
	if err := s.db.First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out, err := toModelPreset(&t)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *MySQLStore) ListPresets() ([]model.Preset, error) {
	var rows []Preset
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPresets(rows)
}

func (s *MySQLStore) ListPresetsByOwner(email string) ([]model.Preset, error) {
	var rows []Preset
	if err := s.db.Where("owner_email = ?", email).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPresets(rows)
}

func (s *MySQLStore) CreatePreset(t *model.Preset) error {
	row, err := fromModelPreset(t)
	if err != nil {
		return err
	}
	return s.db.Create(&row).Error
}

func (s *MySQLStore) UpdatePreset(t *model.Preset) error {
	row, err := fromModelPreset(t)
	if err != nil {
		return err
	}
	return s.db.Save(&row).Error
}

func (s *MySQLStore) DeletePreset(id string) error {
	return s.db.Where("id = ?", id).Delete(&Preset{}).Error
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

// ---------- Plugins ----------

func (s *MySQLStore) GetPlugin(id string) (*model.Plugin, error) {
	var p Plugin
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := toModelPlugin(&p)
	return &out, nil
}

func (s *MySQLStore) ListPlugins() ([]model.Plugin, error) {
	var rows []Plugin
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPlugins(rows), nil
}

func (s *MySQLStore) ListPublishedPlugins() ([]model.Plugin, error) {
	var rows []Plugin
	if err := s.db.Where("is_published = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPlugins(rows), nil
}

func (s *MySQLStore) ListPluginsByAuthor(email string) ([]model.Plugin, error) {
	var rows []Plugin
	if err := s.db.Where("author_email = ?", email).Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPlugins(rows), nil
}

func (s *MySQLStore) SearchPlugins(query, sortBy string) ([]model.Plugin, error) {
	tx := s.db.Where("is_published = ?", true)
	if query != "" {
		like := "%" + query + "%"
		tx = tx.Where("name LIKE ? OR description LIKE ? OR author_email LIKE ?", like, like, like)
	}
	switch sortBy {
	case "Recently Updated":
		tx = tx.Order("updated_at DESC")
	default:
		tx = tx.Order("installs DESC")
	}
	var rows []Plugin
	if err := tx.Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPlugins(rows), nil
}

func (s *MySQLStore) CreatePlugin(p *model.Plugin) error {
	row := fromModelPlugin(p)
	return s.db.Create(&row).Error
}

func (s *MySQLStore) UpdatePlugin(p *model.Plugin) error {
	row := fromModelPlugin(p)
	return s.db.Save(&row).Error
}

func (s *MySQLStore) DeletePlugin(id string) error {
	return s.db.Where("id = ?", id).Delete(&Plugin{}).Error
}

func (s *MySQLStore) IncrementPluginInstalls(id string) error {
	return s.db.Model(&Plugin{}).Where("id = ?", id).Update("installs", gorm.Expr("installs + 1")).Error
}

// ---------- User Plugins (installed) ----------

func (s *MySQLStore) InstallPlugin(userEmail, pluginID string) error {
	row := UserPlugin{
		UserEmail:   userEmail,
		PluginID:    pluginID,
		InstalledAt: time.Now(),
	}
	return s.db.Create(&row).Error
}

func (s *MySQLStore) UninstallPlugin(userEmail, pluginID string) error {
	return s.db.Where("user_email = ? AND plugin_id = ?", userEmail, pluginID).Delete(&UserPlugin{}).Error
}

func (s *MySQLStore) ListInstalledPlugins(userEmail string) ([]model.Plugin, error) {
	var rows []Plugin
	if err := s.db.
		Where("id IN (?)", s.db.Model(&UserPlugin{}).Select("plugin_id").Where("user_email = ?", userEmail)).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return toModelPlugins(rows), nil
}

func (s *MySQLStore) IsPluginInstalled(userEmail, pluginID string) (bool, error) {
	var count int64
	if err := s.db.Model(&UserPlugin{}).Where("user_email = ? AND plugin_id = ?", userEmail, pluginID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
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
	var pc map[string]string
	if len(d.PluginConfig) > 0 {
		_ = json.Unmarshal(d.PluginConfig, &pc)
	}
	return model.Device{
		MAC:          d.MAC,
		Name:         d.Name,
		BoardModelID: d.BoardModelID,
		PluginName:   d.PluginName,
		PresetID:     d.PresetID,
		FirmwareVer:  d.FirmwareVer,
		PluginConfig: pc,
		LastSeen:     d.LastSeen,
		CreatedAt:    d.CreatedAt,
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
	var pc datatypes.JSON
	if d.PluginConfig != nil {
		pc, _ = json.Marshal(d.PluginConfig)
	}
	return Device{
		MAC:          d.MAC,
		Name:         d.Name,
		BoardModelID: d.BoardModelID,
		PluginName:   d.PluginName,
		PresetID:     d.PresetID,
		FirmwareVer:  d.FirmwareVer,
		PluginConfig: pc,
		LastSeen:     d.LastSeen,
		CreatedAt:    d.CreatedAt,
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

func toModelPreset(t *Preset) (model.Preset, error) {
	var vals map[string]string
	if len(t.Values) > 0 {
		if err := json.Unmarshal(t.Values, &vals); err != nil {
			return model.Preset{}, err
		}
	}
	return model.Preset{
		ID:         t.ID,
		Name:       t.Name,
		PluginName: t.PluginName,
		OwnerEmail: t.OwnerEmail,
		IsBuiltIn:  t.IsBuiltIn,
		Values:     vals,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}, nil
}

func toModelPresets(rows []Preset) ([]model.Preset, error) {
	out := make([]model.Preset, len(rows))
	for i := range rows {
		t, err := toModelPreset(&rows[i])
		if err != nil {
			return nil, err
		}
		out[i] = t
	}
	return out, nil
}

func fromModelPreset(t *model.Preset) (Preset, error) {
	vals, err := json.Marshal(t.Values)
	if err != nil {
		return Preset{}, err
	}
	return Preset{
		ID:         t.ID,
		Name:       t.Name,
		PluginName: t.PluginName,
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

func toModelPlugin(p *Plugin) model.Plugin {
	return model.Plugin{
		ID:           p.ID,
		Name:         p.Name,
		Type:         p.Type,
		AuthorEmail:  p.AuthorEmail,
		Description:  p.Description,
		Category:     p.Category,
		LuaSource:    p.LuaSource,
		ConfigFields: json.RawMessage(p.ConfigFields),
		Installs:     p.Installs,
		IsPublished:  p.IsPublished,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func toModelPlugins(rows []Plugin) []model.Plugin {
	out := make([]model.Plugin, len(rows))
	for i := range rows {
		out[i] = toModelPlugin(&rows[i])
	}
	return out
}

func fromModelPlugin(p *model.Plugin) Plugin {
	return Plugin{
		ID:           p.ID,
		Name:         p.Name,
		Type:         p.Type,
		AuthorEmail:  p.AuthorEmail,
		Description:  p.Description,
		Category:     p.Category,
		LuaSource:    p.LuaSource,
		ConfigFields: datatypes.JSON(p.ConfigFields),
		Installs:     p.Installs,
		IsPublished:  p.IsPublished,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
