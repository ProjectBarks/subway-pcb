package bolt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	bbolt "go.etcd.io/bbolt"
)

var (
	bucketDevices      = []byte("devices")
	bucketDeviceAccess = []byte("device_access")
	bucketThemes       = []byte("themes")
	bucketUsers        = []byte("users")
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
		for _, b := range [][]byte{bucketDevices, bucketDeviceAccess, bucketThemes, bucketUsers} {
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

func (s *BoltStore) UpdateDeviceLastSeen(mac string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketDevices)
		v := b.Get([]byte(mac))
		if v == nil {
			return fmt.Errorf("bolt: device %q not found", mac)
		}
		var d model.Device
		if err := json.Unmarshal(v, &d); err != nil {
			return err
		}
		d.LastSeen = time.Now()
		data, err := json.Marshal(&d)
		if err != nil {
			return err
		}
		return b.Put([]byte(mac), data)
	})
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
// Themes
// ---------------------------------------------------------------------------

func (s *BoltStore) GetTheme(id string) (*model.Theme, error) {
	var t model.Theme
	found, err := s.get(bucketThemes, id, &t)
	if err != nil || !found {
		return nil, err
	}
	return &t, nil
}

func (s *BoltStore) ListThemes() ([]model.Theme, error) {
	var themes []model.Theme
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketThemes)
		return b.ForEach(func(_, v []byte) error {
			var t model.Theme
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			themes = append(themes, t)
			return nil
		})
	})
	return themes, err
}

func (s *BoltStore) ListThemesByOwner(email string) ([]model.Theme, error) {
	var themes []model.Theme
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketThemes)
		return b.ForEach(func(_, v []byte) error {
			var t model.Theme
			if err := json.Unmarshal(v, &t); err != nil {
				return err
			}
			if t.OwnerEmail == email {
				themes = append(themes, t)
			}
			return nil
		})
	})
	return themes, err
}

func (s *BoltStore) CreateTheme(t *model.Theme) error {
	return s.put(bucketThemes, t.ID, t)
}

func (s *BoltStore) UpdateTheme(t *model.Theme) error {
	return s.put(bucketThemes, t.ID, t)
}

func (s *BoltStore) DeleteTheme(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bucketThemes).Delete([]byte(id))
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
