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
	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"google.golang.org/protobuf/proto"
)

// Route color table (RGB)
var routeColors = map[pb.Route][3]uint8{
	pb.Route_ROUTE_1:  {238, 53, 46},
	pb.Route_ROUTE_2:  {238, 53, 46},
	pb.Route_ROUTE_3:  {238, 53, 46},
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
const globalBrightness = 20.0 / 255.0

// LEDMap maps flat LED index -> station ID
type LEDMap struct {
	stationIDs []string // index -> station_id (len=476)
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

	// Build flat array: for each strip in order, for each pixel in order
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
// It caches the rendered frame and only re-renders when the aggregator updates.
type PixelRenderer struct {
	ledMap   *LEDMap
	seq      uint32
	cached   []byte // cached protobuf bytes
	cachedAt time.Time
	mu       sync.Mutex
}

// NewPixelRenderer creates a renderer with the given LED map
func NewPixelRenderer(ledMap *LEDMap) *PixelRenderer {
	return &PixelRenderer{ledMap: ledMap}
}

// GetFrame returns the cached frame, re-rendering only when aggregator has new data.
func (pr *PixelRenderer) GetFrame(agg *mta.Aggregator) ([]byte, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Cache for 5 seconds — MTA feeds only update every ~15s anyway
	if pr.cached != nil && time.Since(pr.cachedAt) < 5*time.Second {
		return pr.cached, nil
	}

	// Re-render
	trains := agg.GetStationTrains()
	pr.seq++

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

		pixels[i*3+0] = uint8(float64(color[0]) * globalBrightness)
		pixels[i*3+1] = uint8(float64(color[1]) * globalBrightness)
		pixels[i*3+2] = uint8(float64(color[2]) * globalBrightness)
	}

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
	log.Printf("pixels: rendered frame seq=%d, %d stations active", pr.seq, len(trains))
	return data, nil
}

// handlePixels serves the pre-rendered PixelFrame
func (s *Server) handlePixels(w http.ResponseWriter, r *http.Request) {
	log.Printf("api: %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.pixelRenderer == nil {
		http.Error(w, "pixel renderer not configured", http.StatusServiceUnavailable)
		return
	}

	data, err := s.pixelRenderer.GetFrame(s.aggregator)
	if err != nil {
		log.Printf("api: pixel render error: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "info" {
		// Debug: return info about the pixel data
		w.Header().Set("Content-Type", "application/json")
		nonZero := 0
		for i := 0; i < len(data); i++ {
			if data[i] != 0 {
				nonZero++
			}
		}
		fmt.Fprintf(w, `{"protobuf_bytes":%d,"led_count":%d,"pixel_bytes":%d}`,
			len(data), totalLEDs, totalLEDs*3)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
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
