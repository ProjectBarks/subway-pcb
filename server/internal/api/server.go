package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ProjectBarks/subway-pcb/server/internal/mta"
	"google.golang.org/protobuf/encoding/protojson"
)

// Server is the HTTP API server for SubwayState.
type Server struct {
	aggregator    *mta.Aggregator
	pixelRenderer *PixelRenderer
	startTime     time.Time
	mux           *http.ServeMux
}

// NewServer creates a new API server backed by the given aggregator.
// If visualizerPath is non-empty, GET / will serve that HTML file.
func NewServer(aggregator *mta.Aggregator, pixelRenderer *PixelRenderer, visualizerPath string) *Server {
	s := &Server{
		aggregator:    aggregator,
		pixelRenderer: pixelRenderer,
		startTime:     time.Now(),
		mux:           http.NewServeMux(),
	}
	s.mux.HandleFunc("/api/v1/state", s.handleState)
	s.mux.HandleFunc("/api/v1/pixels", s.handlePixels)
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/board.jpg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "board.jpg")
	})
	s.mux.HandleFunc("/board.svg", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "board.svg")
	})
	s.mux.HandleFunc("/board_paths.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "board_paths.json")
	})
	if visualizerPath != "" {
		vp := visualizerPath
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, vp)
		})
		log.Printf("api: serving visualizer at / from %s", visualizerPath)
	}
	return s
}

// Handler returns the http.Handler for this server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// handleState serves the current SubwayState.
// GET /api/v1/state          -> binary protobuf
// GET /api/v1/state?format=json -> JSON (for debugging)
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	log.Printf("api: %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")

	if format == "json" {
		state := s.aggregator.GetState()
		if state == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
			return
		}

		opts := protojson.MarshalOptions{
			EmitUnpopulated: true,
			UseProtoNames:   true,
		}
		data, err := opts.Marshal(state)
		if err != nil {
			log.Printf("api: JSON marshal error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	// Default: binary protobuf.
	data := s.aggregator.GetStateBytes()
	if data == nil {
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// healthResponse is the JSON shape for the /health endpoint.
type healthResponse struct {
	Status       string  `json:"status"`
	Uptime       string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	LastUpdate   string  `json:"last_update"`
	StationCount int     `json:"station_count"`
}

// handleHealth serves a JSON health check.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(s.startTime)
	lastUpdate := s.aggregator.LastUpdate()

	lastUpdateStr := "never"
	if !lastUpdate.IsZero() {
		lastUpdateStr = fmt.Sprintf("%ds ago", int(time.Since(lastUpdate).Seconds()))
	}

	resp := healthResponse{
		Status:        "ok",
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: uptime.Seconds(),
		LastUpdate:    lastUpdateStr,
		StationCount:  s.aggregator.StationCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
