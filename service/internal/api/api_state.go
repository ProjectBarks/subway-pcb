package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// --- State & Health ---

type (
	JSONTrain struct {
		Route  string `json:"route"`
		Status string `json:"status"`
	}

	JSONStation struct {
		StopID string      `json:"stop_id"`
		Trains []JSONTrain `json:"trains"`
	}

	JSONStateResp struct {
		Timestamp uint64        `json:"timestamp"`
		Stations  []JSONStation `json:"stations"`
	}
)

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stations := s.aggregator.GetStations()
	ts := s.aggregator.GetTimestamp()

	stationList := make([]JSONStation, 0, len(stations))
	for _, st := range stations {
		trains := make([]JSONTrain, 0, len(st.Trains))
		for _, t := range st.Trains {
			trains = append(trains, JSONTrain{
				Route:  t.Route,
				Status: t.Status.String(),
			})
		}
		stationList = append(stationList, JSONStation{
			StopID: st.StopId,
			Trains: trains,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(JSONStateResp{
		Timestamp: ts,
		Stations:  stationList,
	})
}

type HeathResponse struct {
	Status        string  `json:"status"`
	Uptime        string  `json:"uptime"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	LastUpdate    string  `json:"last_update"`
	StationCount  int     `json:"station_count"`
}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HeathResponse{
		Status:        "ok",
		Uptime:        uptime.Round(time.Second).String(),
		UptimeSeconds: uptime.Seconds(),
		LastUpdate:    lastUpdateStr,
		StationCount:  s.aggregator.StationCount(),
	})
}
