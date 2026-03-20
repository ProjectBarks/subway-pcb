package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// BoardManifest is the on-disk JSON representation of a board model.
type BoardManifest struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Version       int           `json:"version"`
	LEDCount      int           `json:"ledCount"`
	Strips        []int         `json:"strips"`
	Model         string        `json:"model"`
	Camera        BoardCamera   `json:"camera"`
	DefaultPlugin string        `json:"defaultPlugin"`
	DefaultPreset string        `json:"defaultPreset"`
	Features      []string      `json:"features"`
	LEDPositions  []LEDPosition `json:"ledPositions"`
}

// BoardCamera holds camera settings for the 3D viewer.
type BoardCamera struct {
	FOV      int `json:"fov"`
	Distance int `json:"distance"`
}

// LEDPosition is a single LED's 3D position and station mapping.
type LEDPosition struct {
	Ref       string  `json:"ref"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	Angle     float64 `json:"angle"`
	Index     int     `json:"index"`
	StationID string  `json:"stationId"`
}

// BoardData holds a loaded board manifest plus derived data.
type BoardData struct {
	Manifest   BoardManifest
	Dir        string   // path to the board version directory (for GLB serving)
	StationIDs []string // flat array: LED index -> station ID, derived from ledPositions + strips
}

// LoadAllBoards walks boardsDir for */v*/board.json files and loads each board.
// Returns a map keyed by "boardID/vN" (e.g. "nyc-subway/v1").
func LoadAllBoards(boardsDir string) (map[string]*BoardData, error) {
	boards := make(map[string]*BoardData)

	matches, err := filepath.Glob(filepath.Join(boardsDir, "*", "v*", "board.json"))
	if err != nil {
		return nil, fmt.Errorf("glob boards: %w", err)
	}

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var manifest BoardManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		dir := filepath.Dir(path)
		key := manifest.ID + "/v" + strconv.Itoa(manifest.Version)

		stationIDs := deriveStationIDs(manifest.LEDPositions, manifest.Strips, manifest.LEDCount)

		boards[key] = &BoardData{
			Manifest:   manifest,
			Dir:        dir,
			StationIDs: stationIDs,
		}

		log.Printf("board: loaded %s (%s, %d LEDs, %d strips)", key, manifest.Name, manifest.LEDCount, len(manifest.Strips))
	}

	if len(boards) == 0 {
		return nil, fmt.Errorf("no boards found in %s", boardsDir)
	}

	return boards, nil
}

// deriveStationIDs builds a flat station ID array from LED positions and strip sizes.
// The flat index is determined by walking strips in order: strip 0 pixels 0..N-1,
// strip 1 pixels 0..M-1, etc. Each LED position's strip and pixel offset is derived
// from its Index field and the strip sizes.
func deriveStationIDs(positions []LEDPosition, strips []int, ledCount int) []string {
	ids := make([]string, ledCount)

	// Build a lookup: flat index -> station ID from positions
	for _, pos := range positions {
		flatIdx := pos.Index
		if flatIdx >= 0 && flatIdx < ledCount && pos.StationID != "" {
			ids[flatIdx] = pos.StationID
		}
	}

	return ids
}

// HasFeature checks if a board has a specific feature.
func (b *BoardData) HasFeature(feature string) bool {
	for _, f := range b.Manifest.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// HasAllFeatures checks if a board has all the specified features.
func (b *BoardData) HasAllFeatures(features []string) bool {
	for _, f := range features {
		if !b.HasFeature(f) {
			return false
		}
	}
	return true
}

// BoardModelKey returns the composite key for looking up a board (e.g. "nyc-subway/v1").
// Falls back to "nyc-subway/v1" for empty input (backward compat).
func BoardModelKey(boardModelID string) string {
	if boardModelID == "" {
		return "nyc-subway/v1"
	}
	return boardModelID
}

// BoardURLPath returns the URL path for serving a board's static files.
// e.g. "nyc-subway/v1" -> "/static/dist/boards/nyc-subway/v1/board.json"
func BoardURLPath(boardModelID string) string {
	key := BoardModelKey(boardModelID)
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return "/static/dist/boards/" + parts[0] + "/" + parts[1] + "/board.json"
	}
	return "/static/dist/boards/" + key + "/board.json"
}
