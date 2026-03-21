package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// --- State & Health ---

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stations := s.aggregator.GetStations()
	ts := s.aggregator.GetTimestamp()

	type stateJSONTrain struct {
		Route  string `json:"route"`
		Status string `json:"status"`
	}
	type stateJSONStation struct {
		StopID string           `json:"stop_id"`
		Trains []stateJSONTrain `json:"trains"`
	}

	stationList := make([]stateJSONStation, 0, len(stations))
	for _, st := range stations {
		trains := make([]stateJSONTrain, 0, len(st.Trains))
		for _, t := range st.Trains {
			trains = append(trains, stateJSONTrain{
				Route:  t.Route,
				Status: t.Status.String(),
			})
		}
		stationList = append(stationList, stateJSONStation{
			StopID: st.StopId,
			Trains: trains,
		})
	}

	resp := struct {
		Timestamp uint64             `json:"timestamp"`
		Stations  []stateJSONStation `json:"stations"`
	}{
		Timestamp: ts,
		Stations:  stationList,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type healthResponse struct {
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

// --- Users (Admin) ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, users)
}
