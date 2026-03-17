package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/server/gen/subwaypb"
	"github.com/ProjectBarks/subway-pcb/server/internal/mode"
	"github.com/ProjectBarks/subway-pcb/server/internal/model"
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"github.com/ProjectBarks/subway-pcb/server/internal/store"
	"google.golang.org/protobuf/proto"
)

// Route color table (RGB) — kept as fallback for firmware without device tracking
var routeColors = map[pb.Route][3]uint8{
	pb.Route_ROUTE_1:  {255, 0, 0},
	pb.Route_ROUTE_2:  {255, 0, 0},
	pb.Route_ROUTE_3:  {255, 0, 0},
	pb.Route_ROUTE_4:  {0, 147, 60},
	pb.Route_ROUTE_5:  {0, 147, 60},
	pb.Route_ROUTE_6:  {0, 147, 60},
	pb.Route_ROUTE_7:  {185, 51, 173},
	pb.Route_ROUTE_A:  {0, 57, 166},
	pb.Route_ROUTE_B:  {255, 99, 25},
	pb.Route_ROUTE_C:  {0, 57, 166},
	pb.Route_ROUTE_D:  {255, 99, 25},
	pb.Route_ROUTE_E:  {0, 57, 166},
	pb.Route_ROUTE_F:  {255, 99, 25},
	pb.Route_ROUTE_G:  {108, 190, 69},
	pb.Route_ROUTE_J:  {153, 102, 51},
	pb.Route_ROUTE_L:  {167, 169, 172},
	pb.Route_ROUTE_M:  {255, 99, 25},
	pb.Route_ROUTE_N:  {252, 204, 10},
	pb.Route_ROUTE_Q:  {252, 204, 10},
	pb.Route_ROUTE_R:  {252, 204, 10},
	pb.Route_ROUTE_W:  {252, 204, 10},
	pb.Route_ROUTE_Z:  {153, 102, 51},
	pb.Route_ROUTE_S:  {128, 129, 131},
	pb.Route_ROUTE_FS: {128, 129, 131},
	pb.Route_ROUTE_GS: {128, 129, 131},
	pb.Route_ROUTE_SI: {0, 57, 166},
}

// Strip sizes in order
var stripSizes = [9]int{97, 102, 55, 81, 70, 21, 22, 19, 11}

const totalLEDs = 478

// LEDMap maps flat LED index -> station ID
type LEDMap struct {
	stationIDs []string // index -> station_id (len=478)
}

// StationIDs returns the flat station ID mapping for use by the mode system.
func (m *LEDMap) StationIDs() []string {
	return m.stationIDs
}

// LoadLEDMap loads the led_map.json file
func LoadLEDMap(path string) (*LEDMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read led_map.json: %w", err)
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse led_map.json: %w", err)
	}

	ids := make([]string, totalLEDs)
	offset := 0
	for strip := 0; strip < 9; strip++ {
		for pixel := 0; pixel < stripSizes[strip]; pixel++ {
			key := fmt.Sprintf("%d,%d", strip, pixel)
			if sid, ok := raw[key]; ok {
				ids[offset] = sid
			}
			offset++
		}
	}

	log.Printf("ledmap: loaded %d entries from %s", len(raw), path)
	return &LEDMap{stationIDs: ids}, nil
}

// PixelRenderer generates PixelFrame protobuf from aggregator state.
type PixelRenderer struct {
	ledMap      *LEDMap
	store       store.Store
	modes       *mode.Registry
	seq         uint32
	cached      []byte
	cachedAt    time.Time
	cachedKey   string
	mu          sync.Mutex
}

// NewPixelRenderer creates a renderer with the given LED map.
func NewPixelRenderer(ledMap *LEDMap) *PixelRenderer {
	return &PixelRenderer{ledMap: ledMap}
}

// SetDeps sets the store and mode registry (called after construction to avoid circular init).
func (pr *PixelRenderer) SetDeps(s store.Store, m *mode.Registry) {
	pr.store = s
	pr.modes = m
}

// Invalidate clears the cached frame.
func (pr *PixelRenderer) Invalidate() {
	pr.mu.Lock()
	pr.cached = nil
	pr.mu.Unlock()
}

// GetFrame returns the rendered frame for the default device (backward compat).
func (pr *PixelRenderer) GetFrame(agg *mta.Aggregator) ([]byte, error) {
	return pr.GetFrameForDevice(agg, "")
}

// GetFrameForDevice returns a rendered frame for a specific device, or the default if mac is empty.
func (pr *PixelRenderer) GetFrameForDevice(agg *mta.Aggregator, mac string) ([]byte, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	cacheKey := mac
	if pr.cached != nil && time.Since(pr.cachedAt) < 5*time.Second && pr.cachedKey == cacheKey {
		return pr.cached, nil
	}

	var pixels []byte
	var err error

	// Try mode-based rendering if store is available
	if pr.store != nil && pr.modes != nil && mac != "" {
		pixels, err = pr.renderWithMode(agg, mac)
	}

	// Fallback to classic rendering
	if pixels == nil || err != nil {
		pixels = pr.renderClassic(agg)
	}

	pr.seq++
	frame := &pb.PixelFrame{
		Timestamp: uint64(time.Now().Unix()),
		Sequence:  pr.seq,
		LedCount:  totalLEDs,
		Pixels:    pixels,
	}

	data, err := proto.Marshal(frame)
	if err != nil {
		return nil, err
	}

	pr.cached = data
	pr.cachedAt = time.Now()
	pr.cachedKey = cacheKey
	log.Printf("pixels: rendered frame seq=%d for device=%s", pr.seq, mac)
	return data, nil
}

// renderWithMode renders using the device's configured mode and theme.
func (pr *PixelRenderer) renderWithMode(agg *mta.Aggregator, mac string) ([]byte, error) {
	device, err := pr.store.GetDevice(mac)
	if err != nil || device == nil {
		return nil, fmt.Errorf("device not found: %s", mac)
	}

	modeName := device.Mode
	if modeName == "" {
		modeName = "track"
	}

	m, ok := pr.modes.Get(modeName)
	if !ok {
		return nil, fmt.Errorf("unknown mode: %s", modeName)
	}

	// Build config: start with mode field defaults, layer theme values, then device overrides
	config := make(map[string]string)
	for _, f := range m.ConfigFields() {
		config[f.Key] = f.Default
	}
	if device.ThemeID != "" {
		theme, _ := pr.store.GetTheme(device.ThemeID)
		if theme != nil {
			for k, v := range theme.Values {
				config[k] = v
			}
		}
	}
	for k, v := range device.ModeConfig {
		config[k] = v
	}

	ctx := mode.RenderContext{
		Aggregator: agg,
		StationIDs: pr.ledMap.stationIDs,
		Device:     device,
		Config:     config,
		TotalLEDs:  totalLEDs,
	}

	return m.Render(ctx)
}

// renderClassic is the original rendering logic — no mode system, hardcoded colors.
func (pr *PixelRenderer) renderClassic(agg *mta.Aggregator) []byte {
	trains := agg.GetStationTrains()
	pixels := make([]byte, totalLEDs*3)

	for i := 0; i < totalLEDs; i++ {
		sid := pr.ledMap.stationIDs[i]
		if sid == "" {
			continue
		}

		info, active := trains[sid]
		if !active {
			continue
		}

		color, ok := routeColors[info.Route]
		if !ok {
			continue
		}

		pixels[i*3+0] = color[0]
		pixels[i*3+1] = color[1]
		pixels[i*3+2] = color[2]
	}

	return pixels
}

// handlePixels serves the pre-rendered PixelFrame.
func (s *Server) handlePixels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.pixelRenderer == nil {
		http.Error(w, "pixel renderer not configured", http.StatusServiceUnavailable)
		return
	}

	// Support per-device rendering
	mac := r.URL.Query().Get("device")
	if mac == "" {
		// Check X-Device-ID header (firmware sends this)
		mac = r.Header.Get("X-Device-ID")
	}

	// Auto-register device if it has an ID
	if mac != "" && s.store != nil {
		s.autoRegisterDevice(mac, r)
	}

	var data []byte
	var err error

	if mac != "" {
		data, err = s.pixelRenderer.GetFrameForDevice(s.aggregator, mac)
	} else {
		data, err = s.pixelRenderer.GetFrame(s.aggregator)
	}

	if err != nil {
		log.Printf("api: pixel render error: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "info" {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"protobuf_bytes":%d,"led_count":%d,"pixel_bytes":%d,"device":"%s"}`,
			len(data), totalLEDs, totalLEDs*3, mac)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// autoRegisterDevice creates or updates a device record when it connects.
func (s *Server) autoRegisterDevice(mac string, r *http.Request) {
	existing, _ := s.store.GetDevice(mac)
	if existing != nil {
		// Just update last seen
		s.store.UpdateDeviceLastSeen(mac)
		return
	}

	// Auto-register new device
	device := &model.Device{
		MAC:       mac,
		Mode:      "track",
		ThemeID:   "classic-mta",
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}

	// Try to get firmware version from header
	if fwVer := r.Header.Get("X-Firmware-Version"); fwVer != "" {
		device.FirmwareVer = fwVer
	}

	if err := s.store.UpsertDevice(device); err != nil {
		log.Printf("api: failed to auto-register device %s: %v", mac, err)
	} else {
		log.Printf("api: auto-registered new device %s", mac)
	}
}

// parseLEDMapDimension parses "strip,pixel" format
func parseLEDMapDimension(key string) (int, int, error) {
	parts := strings.SplitN(key, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad key: %s", key)
	}
	strip, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	pixel, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return strip, pixel, nil
}
