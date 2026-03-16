package mta

import (
	"log"
	"sort"
	"sync"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/server/gen/subwaypb"
	"google.golang.org/protobuf/proto"
)

// trainUpdate represents a single train observation from a feed.
type trainUpdate struct {
	StopID string
	Route  pb.Route
	Status pb.TrainStatus
}

// feedSnapshot holds the complete state from one MTA feed poll.
// When a feed updates, its entire snapshot is replaced atomically.
type feedSnapshot struct {
	updates []trainUpdate
}

// stationTrain is a unique train at a station (route+status).
type stationTrain struct {
	Route  pb.Route
	Status pb.TrainStatus
}

// Aggregator collects feed data from all pollers and builds a SubwayState.
// It uses per-feed replacement: each feed's data is kept until the next
// update from that same feed, eliminating expiry gaps.
type Aggregator struct {
	mu         sync.RWMutex
	feeds      map[string]*feedSnapshot // keyed by feed name
	sequence   uint32
	lastUpdate time.Time

	// Cached serialized protobuf.
	cachedBytes []byte
	cachedState *pb.SubwayState
}

// NewAggregator creates a new Aggregator.
func NewAggregator(_ time.Duration) *Aggregator {
	return &Aggregator{
		feeds: make(map[string]*feedSnapshot),
	}
}

// IngestFeed replaces all data from a named feed with new updates.
// This is atomic per-feed: old data from this feed is fully replaced.
func (a *Aggregator) IngestFeed(feedName string, updates []trainUpdate) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.feeds[feedName] = &feedSnapshot{updates: updates}
	a.rebuildLocked()
}

// Ingest is a compatibility shim — uses "default" as feed name.
func (a *Aggregator) Ingest(updates []trainUpdate) {
	a.IngestFeed("default", updates)
}

// rebuildLocked merges all feed snapshots and rebuilds the cached state.
func (a *Aggregator) rebuildLocked() {
	now := time.Now()
	a.sequence++
	a.lastUpdate = now

	// Merge all feeds into a station -> trains map.
	type trainKey struct {
		Route  pb.Route
		Status pb.TrainStatus
	}
	stationTrains := make(map[string]map[trainKey]struct{})

	for _, snap := range a.feeds {
		for _, u := range snap.updates {
			parentID := NormalizeStopID(u.StopID)
			if parentID == "" {
				continue
			}
			if _, ok := stationTrains[parentID]; !ok {
				stationTrains[parentID] = make(map[trainKey]struct{})
			}
			stationTrains[parentID][trainKey{Route: u.Route, Status: u.Status}] = struct{}{}
		}
	}

	// Build protobuf message.
	state := &pb.SubwayState{
		Timestamp: uint64(now.Unix()),
		Sequence:  a.sequence,
	}

	stopIDs := make([]string, 0, len(stationTrains))
	for stopID := range stationTrains {
		stopIDs = append(stopIDs, stopID)
	}
	sort.Strings(stopIDs)

	for _, stopID := range stopIDs {
		trains := make([]*pb.Train, 0, len(stationTrains[stopID]))
		for key := range stationTrains[stopID] {
			trains = append(trains, &pb.Train{
				Route:  key.Route,
				Status: key.Status,
			})
		}
		sort.Slice(trains, func(i, j int) bool {
			if trains[i].Route != trains[j].Route {
				return trains[i].Route < trains[j].Route
			}
			return trains[i].Status < trains[j].Status
		})
		if len(trains) > 4 {
			trains = trains[:4]
		}
		state.Stations = append(state.Stations, &pb.Station{
			StopId: stopID,
			Trains: trains,
		})
	}

	a.cachedState = state

	data, err := proto.Marshal(state)
	if err != nil {
		log.Printf("ERROR: failed to marshal SubwayState: %v", err)
		return
	}
	a.cachedBytes = data
}

// GetState returns the current cached SubwayState protobuf message.
func (a *Aggregator) GetState() *pb.SubwayState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cachedState
}

// GetStateBytes returns the current cached serialized SubwayState.
func (a *Aggregator) GetStateBytes() []byte {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cachedBytes
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
	if a.cachedState == nil {
		return 0
	}
	return len(a.cachedState.Stations)
}

// StationTrainInfo holds the highest-priority train at a station for pixel rendering.
type StationTrainInfo struct {
	Route  pb.Route
	Status pb.TrainStatus
}

// GetStationTrains returns the highest-priority train for each active station.
func (a *Aggregator) GetStationTrains() map[string]StationTrainInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.cachedState == nil {
		return nil
	}

	result := make(map[string]StationTrainInfo, len(a.cachedState.Stations))
	for _, station := range a.cachedState.Stations {
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
