package mta

import (
	"sort"
	"sync"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
)

// trainUpdate represents a single train observation from a feed.
type trainUpdate struct {
	StopID string
	Route  string
	Status pb.TrainStatus
	TripID string // unique per physical train
}

// feedSnapshot holds the complete state from one MTA feed poll.
type feedSnapshot struct {
	updates   []trainUpdate
	timestamp time.Time
}

// Aggregator collects feed data from all pollers and builds station state.
type Aggregator struct {
	mu         sync.RWMutex
	feeds      map[string]*feedSnapshot
	sequence   uint32
	lastUpdate time.Time

	cachedStations []*pb.Station
	cachedTimestamp uint64
}

// NewAggregator creates a new Aggregator.
func NewAggregator(_ time.Duration) *Aggregator {
	return &Aggregator{
		feeds: make(map[string]*feedSnapshot),
	}
}

// IngestFeed replaces all data from a named feed with new updates.
func (a *Aggregator) IngestFeed(feedName string, updates []trainUpdate) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.feeds[feedName] = &feedSnapshot{updates: updates, timestamp: time.Now()}
	a.rebuildLocked()
}

// Ingest is a compatibility shim.
func (a *Aggregator) Ingest(updates []trainUpdate) {
	a.IngestFeed("default", updates)
}

// rebuildLocked merges all feed snapshots into a clean state.
// Strategy: track each physical train by TripID. Only show trains
// that are STOPPED_AT a station. One train per route per station max.
func (a *Aggregator) rebuildLocked() {
	now := time.Now()
	a.sequence++
	a.lastUpdate = now

	// Deduplicate by TripID — each physical train appears once.
	// If same TripID appears in multiple feeds (shouldn't happen), latest wins.
	type trainInfo struct {
		StopID string
		Route  string
		Status pb.TrainStatus
	}
	trainsByTrip := make(map[string]trainInfo)

	for _, snap := range a.feeds {
		for _, u := range snap.updates {
			parentID := NormalizeStopID(u.StopID)
			if parentID == "" {
				continue
			}

			// Only show trains actually at a station
			if u.Status != pb.TrainStatus_STOPPED_AT {
				continue
			}

			if u.TripID != "" {
				// Track by trip — one entry per physical train
				trainsByTrip[u.TripID] = trainInfo{
					StopID: parentID,
					Route:  u.Route,
					Status: u.Status,
				}
			} else {
				// No trip ID — use a synthetic key
				key := parentID + ":" + u.Route
				trainsByTrip[key] = trainInfo{
					StopID: parentID,
					Route:  u.Route,
					Status: u.Status,
				}
			}
		}
	}

	// Now build station -> best route map (one route per station max for cleaner display)
	stationBest := make(map[string]map[string]pb.TrainStatus)

	for _, t := range trainsByTrip {
		if _, ok := stationBest[t.StopID]; !ok {
			stationBest[t.StopID] = make(map[string]pb.TrainStatus)
		}
		existing, exists := stationBest[t.StopID][t.Route]
		if !exists || statusPriority(t.Status) > statusPriority(existing) {
			stationBest[t.StopID][t.Route] = t.Status
		}
	}

	// Build station list.
	stopIDs := make([]string, 0, len(stationBest))
	for stopID := range stationBest {
		stopIDs = append(stopIDs, stopID)
	}
	sort.Strings(stopIDs)

	stations := make([]*pb.Station, 0, len(stopIDs))
	for _, stopID := range stopIDs {
		routes := stationBest[stopID]
		trains := make([]*pb.Train, 0, len(routes))
		for route, status := range routes {
			trains = append(trains, &pb.Train{
				Route:  route,
				Status: status,
			})
		}
		sort.Slice(trains, func(i, j int) bool {
			return trains[i].Route < trains[j].Route
		})
		// Limit to 2 trains per station for cleaner display
		if len(trains) > 2 {
			trains = trains[:2]
		}
		stations = append(stations, &pb.Station{
			StopId: stopID,
			Trains: trains,
		})
	}

	a.cachedStations = stations
	a.cachedTimestamp = uint64(now.Unix())
}

// GetStations returns the current cached station list.
func (a *Aggregator) GetStations() []*pb.Station {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cachedStations
}

// GetTimestamp returns the cached timestamp.
func (a *Aggregator) GetTimestamp() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cachedTimestamp
}

// LastUpdate returns the time of the last state rebuild.
func (a *Aggregator) LastUpdate() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastUpdate
}

// StationCount returns the number of active stations.
func (a *Aggregator) StationCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.cachedStations)
}

// StationTrainInfo holds the highest-priority train at a station for pixel rendering.
type StationTrainInfo struct {
	Route  string
	Status pb.TrainStatus
}

// GetStationTrains returns the highest-priority train for each active station.
func (a *Aggregator) GetStationTrains() map[string]StationTrainInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.cachedStations == nil {
		return nil
	}

	result := make(map[string]StationTrainInfo, len(a.cachedStations))
	for _, station := range a.cachedStations {
		var best StationTrainInfo
		bestPriority := -1
		for _, train := range station.Trains {
			p := statusPriority(train.Status)
			if p > bestPriority {
				bestPriority = p
				best = StationTrainInfo{Route: train.Route, Status: train.Status}
			}
		}
		if bestPriority >= 0 {
			result[station.StopId] = best
		}
	}
	return result
}

func statusPriority(s pb.TrainStatus) int {
	switch s {
	case pb.TrainStatus_STOPPED_AT:
		return 3
	case pb.TrainStatus_INCOMING_AT:
		return 2
	case pb.TrainStatus_IN_TRANSIT_TO:
		return 1
	default:
		return 0
	}
}
