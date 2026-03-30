package bolt

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	bbolt "go.etcd.io/bbolt"
)

var (
	bucketDevices      = []byte("devices")
	bucketDeviceAccess = []byte("device_access")
	bucketPresets      = []byte("themes")
	bucketUsers        = []byte("users")
	bucketPlugins      = []byte("plugins")
	bucketUserPlugins  = []byte("user_plugins")
	bucketDiagnostics  = []byte("diagnostics")
)

// BoltStore implements store.Store using an embedded bbolt database.
type BoltStore struct {
	db *bbolt.DB
}

// New opens (or creates) a bbolt database at path and ensures all required
// buckets exist. The returned BoltStore is ready for use.
func New(path string) (*BoltStore, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("bolt: open db: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketDevices, bucketDeviceAccess, bucketPresets, bucketUsers, bucketPlugins, bucketUserPlugins, bucketDiagnostics} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return fmt.Errorf("bolt: create bucket %s: %w", b, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltStore{db: db}, nil
}

// ---------------------------------------------------------------------------
// Devices
// ---------------------------------------------------------------------------

func (s *BoltStore) GetDevice(mac string) (*model.Device, error) {
	var d model.Device
	found, err := s.get(bucketDevices, mac, &d)
	if err != nil || !found {
		return nil, err
	}
	return &d, nil
}

func (s *BoltStore) ListDevices() ([]model.Device, error) {
	var devices []model.Device
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketDevices)
		return b.ForEach(func(_, v []byte) error {
			var d model.Device
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			devices = append(devices, d)
			return nil
		})
	})
	return devices, err
}

func (s *BoltStore) ListDevicesByUser(email string) ([]model.Device, error) {
	accessList, err := s.ListAccessByUser(email)
	if err != nil {
		return nil, err
	}

	var devices []model.Device
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketDevices)
		for _, a := range accessList {
			v := b.Get([]byte(a.MAC))
			if v == nil {
				continue
			}
			var d model.Device
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			devices = append(devices, d)
		}
		return nil
	})
	return devices, err
}

func (s *BoltStore) UpsertDevice(d *model.Device) error {
	return s.put(bucketDevices, d.MAC, d)
}

// ---------------------------------------------------------------------------
// Access
// ---------------------------------------------------------------------------

func accessKey(mac, email string) string {
	return mac + ":" + email
}

func (s *BoltStore) GrantAccess(a *model.DeviceAccess) error {
	return s.put(bucketDeviceAccess, accessKey(a.MAC, a.UserEmail), a)
}

func (s *BoltStore) RevokeAccess(mac, email string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketDeviceAccess).Delete([]byte(accessKey(mac, email)))
	})
}

func (s *BoltStore) ListAccessByDevice(mac string) ([]model.DeviceAccess, error) {
	var results []model.DeviceAccess
	prefix := mac + ":"
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketDeviceAccess)
		return b.ForEach(func(k, v []byte) error {
			if strings.HasPrefix(string(k), prefix) {
				var a model.DeviceAccess
				if err := json.Unmarshal(v, &a); err != nil {
					return err
				}
				results = append(results, a)
			}
			return nil
		})
	})
	return results, err
}

func (s *BoltStore) ListAccessByUser(email string) ([]model.DeviceAccess, error) {
	var results []model.DeviceAccess
	suffix := ":" + email
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketDeviceAccess)
		return b.ForEach(func(k, v []byte) error {
			if strings.HasSuffix(string(k), suffix) {
				var a model.DeviceAccess
				if err := json.Unmarshal(v, &a); err != nil {
					return err
				}
				results = append(results, a)
			}
			return nil
		})
	})
	return results, err
}

func (s *BoltStore) HasAccess(mac, email string) (bool, error) {
	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucketDeviceAccess).Get([]byte(accessKey(mac, email)))
		found = v != nil
		return nil
	})
	return found, err
}

// ---------------------------------------------------------------------------
// Presets
// ---------------------------------------------------------------------------

func (s *BoltStore) GetPreset(id string) (*model.Preset, error) {
	var t model.Preset
	found, err := s.get(bucketPresets, id, &t)
	if err != nil || !found {
		return nil, err
	}
	return &t, nil
}

func (s *BoltStore) ListPresets() ([]model.Preset, error) {
	var presets []model.Preset
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPresets)
		return b.ForEach(func(_, v []byte) error {
			var t model.Preset
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			presets = append(presets, t)
			return nil
		})
	})
	return presets, err
}

func (s *BoltStore) ListPresetsByOwner(email string) ([]model.Preset, error) {
	var presets []model.Preset
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPresets)
		return b.ForEach(func(_, v []byte) error {
			var t model.Preset
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			if t.OwnerEmail == email {
				presets = append(presets, t)
			}
			return nil
		})
	})
	return presets, err
}

func (s *BoltStore) CreatePreset(t *model.Preset) error {
	return s.put(bucketPresets, t.ID, t)
}

func (s *BoltStore) UpdatePreset(t *model.Preset) error {
	return s.put(bucketPresets, t.ID, t)
}

func (s *BoltStore) DeletePreset(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketPresets).Delete([]byte(id))
	})
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func (s *BoltStore) GetUser(email string) (*model.User, error) {
	var u model.User
	found, err := s.get(bucketUsers, email, &u)
	if err != nil || !found {
		return nil, err
	}
	return &u, nil
}

func (s *BoltStore) UpsertUser(u *model.User) error {
	return s.put(bucketUsers, u.Email, u)
}

func (s *BoltStore) ListUsers() ([]model.User, error) {
	var users []model.User
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketUsers)
		return b.ForEach(func(_, v []byte) error {
			var u model.User
			if err := json.Unmarshal(v, &u); err != nil {
				return err
			}
			users = append(users, u)
			return nil
		})
	})
	return users, err
}

// ---------------------------------------------------------------------------
// Plugins
// ---------------------------------------------------------------------------

func (s *BoltStore) GetPlugin(id string) (*model.Plugin, error) {
	var p model.Plugin
	found, err := s.get(bucketPlugins, id, &p)
	if err != nil || !found {
		return nil, err
	}
	return &p, nil
}

func (s *BoltStore) ListPlugins() ([]model.Plugin, error) {
	var plugins []model.Plugin
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlugins)
		return b.ForEach(func(_, v []byte) error {
			var p model.Plugin
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}
			plugins = append(plugins, p)
			return nil
		})
	})
	return plugins, err
}

func (s *BoltStore) ListPublishedPlugins() ([]model.Plugin, error) {
	var plugins []model.Plugin
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlugins)
		return b.ForEach(func(_, v []byte) error {
			var p model.Plugin
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}
			if p.IsPublished {
				plugins = append(plugins, p)
			}
			return nil
		})
	})
	return plugins, err
}

func (s *BoltStore) ListPluginsByAuthor(email string) ([]model.Plugin, error) {
	var plugins []model.Plugin
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlugins)
		return b.ForEach(func(_, v []byte) error {
			var p model.Plugin
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}
			if p.AuthorEmail == email {
				plugins = append(plugins, p)
			}
			return nil
		})
	})
	return plugins, err
}

func (s *BoltStore) SearchPlugins(query, sortBy string) ([]model.Plugin, error) {
	published, err := s.ListPublishedPlugins()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var filtered []model.Plugin
	for _, p := range published {
		if q == "" ||
			strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Description), q) ||
			strings.Contains(strings.ToLower(p.AuthorEmail), q) {
			filtered = append(filtered, p)
		}
	}

	switch sortBy {
	case "Recently Updated":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
		})
	default: // "Most Popular", "Most Installed"
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Installs > filtered[j].Installs
		})
	}

	return filtered, nil
}

func (s *BoltStore) CreatePlugin(p *model.Plugin) error {
	return s.put(bucketPlugins, p.ID, p)
}

func (s *BoltStore) UpdatePlugin(p *model.Plugin) error {
	return s.put(bucketPlugins, p.ID, p)
}

func (s *BoltStore) DeletePlugin(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketPlugins).Delete([]byte(id))
	})
}

func (s *BoltStore) IncrementPluginInstalls(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketPlugins)
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("bolt: plugin %q not found", id)
		}
		var p model.Plugin
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		p.Installs++
		data, err := json.Marshal(&p)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), data)
	})
}

// ---------------------------------------------------------------------------
// User Plugins (installed)
// ---------------------------------------------------------------------------

func userPluginKey(email, pluginID string) string {
	return email + ":" + pluginID
}

func (s *BoltStore) InstallPlugin(userEmail, pluginID string) error {
	up := model.UserPlugin{
		UserEmail:   userEmail,
		PluginID:    pluginID,
		InstalledAt: time.Now(),
	}
	return s.put(bucketUserPlugins, userPluginKey(userEmail, pluginID), &up)
}

func (s *BoltStore) UninstallPlugin(userEmail, pluginID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketUserPlugins).Delete([]byte(userPluginKey(userEmail, pluginID)))
	})
}

func (s *BoltStore) ListInstalledPlugins(userEmail string) ([]model.Plugin, error) {
	prefix := userEmail + ":"
	var pluginIDs []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketUserPlugins)
		return b.ForEach(func(k, v []byte) error {
			key := string(k)
			if strings.HasPrefix(key, prefix) {
				var up model.UserPlugin
				if err := json.Unmarshal(v, &up); err != nil {
					return err
				}
				pluginIDs = append(pluginIDs, up.PluginID)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	var plugins []model.Plugin
	for _, id := range pluginIDs {
		p, err := s.GetPlugin(id)
		if err != nil {
			return nil, err
		}
		if p != nil {
			plugins = append(plugins, *p)
		}
	}
	return plugins, nil
}

func (s *BoltStore) IsPluginInstalled(userEmail, pluginID string) (bool, error) {
	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucketUserPlugins).Get([]byte(userPluginKey(userEmail, pluginID)))
		found = v != nil
		return nil
	})
	return found, err
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

func (s *BoltStore) SaveDiagnostic(d *model.DeviceDiagnostic) error {
	// Key by device ID — only keep latest per device
	return s.put(bucketDiagnostics, d.DeviceID, d)
}

func (s *BoltStore) GetLatestDiagnostic(deviceID string) (*model.DeviceDiagnostic, error) {
	var d model.DeviceDiagnostic
	found, err := s.get(bucketDiagnostics, deviceID, &d)
	if err != nil || !found {
		return nil, err
	}
	return &d, nil
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func (s *BoltStore) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// get reads a single JSON-encoded value from the given bucket. It returns
// (false, nil) when the key does not exist.
func (s *BoltStore) get(bucket []byte, key string, dest any) (bool, error) {
	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucket).Get([]byte(key))
		if v == nil {
			return nil
		}
		found = true
		return json.Unmarshal(v, dest)
	})
	return found, err
}

// put JSON-encodes value and stores it under key in the given bucket.
func (s *BoltStore) put(bucket []byte, key string, value any) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return tx.Bucket(bucket).Put([]byte(key), data)
	})
}
