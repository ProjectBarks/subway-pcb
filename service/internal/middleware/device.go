package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ProjectBarks/subway-pcb/service/internal/model"
	"github.com/ProjectBarks/subway-pcb/service/internal/store"
)

// BoardDefaults holds the default plugin and preset for a board model.
type BoardDefaults struct {
	DefaultPlugin string
	DefaultPreset string
}

const hardwareKey contextKey = "hardware"

// HardwareFromContext extracts the hardware string from the context.
func HardwareFromContext(ctx context.Context) string {
	s, _ := ctx.Value(hardwareKey).(string)
	return s
}

// DeviceAutoRegister returns middleware that auto-registers devices on first
// contact and updates LastSeen + FirmwareVer for known devices.
func DeviceAutoRegister(s store.Store, boardDefaults map[string]BoardDefaults) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mac := r.Header.Get("X-Device-ID")
			hardware := r.Header.Get("X-Hardware")

			// Store hardware in context for downstream handlers
			ctx := context.WithValue(r.Context(), hardwareKey, hardware)
			r = r.WithContext(ctx)

			if mac == "" {
				next.ServeHTTP(w, r)
				return
			}

			existing, _ := s.GetDevice(mac)
			if existing != nil {
				// Known device — update LastSeen + FirmwareVer in background
				fwVer := r.Header.Get("X-Firmware-Version")
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = ctx
					existing.LastSeen = time.Now()
					if fwVer != "" {
						existing.FirmwareVer = fwVer
					}
					if err := s.UpsertDevice(existing); err != nil {
						log.Printf("middleware: failed to update device %s last seen: %v", mac, err)
					}
				}()
				next.ServeHTTP(w, r)
				return
			}

			// New device — synchronously register with board defaults
			boardModelID := hardware
			if boardModelID == "" {
				boardModelID = "nyc-subway/v1"
			}

			bd := boardDefaults[boardModelID]
			device := &model.Device{
				MAC:          mac,
				BoardModelID: boardModelID,
				PluginName:   bd.DefaultPlugin,
				PresetID:     bd.DefaultPreset,
				LastSeen:     time.Now(),
				CreatedAt:    time.Now(),
			}
			if fwVer := r.Header.Get("X-Firmware-Version"); fwVer != "" {
				device.FirmwareVer = fwVer
			}

			if err := s.UpsertDevice(device); err != nil {
				log.Printf("middleware: failed to auto-register device %s: %v", mac, err)
			} else {
				log.Printf("middleware: auto-registered new device %s (board=%s)", mac, boardModelID)
			}

			next.ServeHTTP(w, r)
		})
	}
}
