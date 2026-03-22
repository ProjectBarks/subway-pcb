package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
)

// Device HTTP header constants.
const (
	HeaderDeviceID        = "X-Device-ID"
	HeaderHardware        = "X-Hardware"
	HeaderFirmwareVersion = "X-Firmware-Version"
)

// BoardDefaults holds the default plugin and preset for a board model.
type BoardDefaults struct {
	DefaultPlugin string
	DefaultPreset string
}

// DeviceAutoRegister returns middleware that auto-registers devices on first
// contact and updates LastSeen + FirmwareVer for known devices.
func DeviceAutoRegister(s store.Store, boardDefaults map[string]BoardDefaults) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mac := r.Header.Get(HeaderDeviceID)
			hardware := r.Header.Get(HeaderHardware)
			fwVer := r.Header.Get(HeaderFirmwareVersion)
			if mac == "" || hardware == "" || fwVer == "" {
				next.ServeHTTP(w, r)
				return
			}

			device, _ := s.GetDevice(mac)
			isNew := device == nil

			if isNew {
				bd := boardDefaults[hardware]
				device = &model.Device{
					MAC:          mac,
					BoardModelID: hardware,
					PluginName:   bd.DefaultPlugin,
					PresetID:     bd.DefaultPreset,
					CreatedAt:    time.Now(),
				}
			}

			device.LastSeen = time.Now()
			device.FirmwareVer = fwVer
			device.BoardModelID = hardware

			upsert := func() {
				if err := s.UpsertDevice(device); err != nil {
					log.Printf("middleware: device upsert error for %s: %v", mac, err)
				} else if isNew {
					log.Printf("middleware: auto-registered new device %s (board=%s)", mac, device.BoardModelID)
				}
			}

			if isNew {
				upsert()
			} else { // for exiting items we can update async
				go upsert()
			}

			next.ServeHTTP(w, r)
		})
	}
}
