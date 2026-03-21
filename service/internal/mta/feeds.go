package mta

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

const mtaBaseURL = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2F"

// feedSuffixes defines the 9 MTA GTFS-RT feed endpoints.
var feedSuffixes = []string{
	"gtfs",      // 1/2/3/4/5/6/7/S
	"gtfs-ace",  // A/C/E
	"gtfs-nqrw", // N/Q/R/W
	"gtfs-bdfm", // B/D/F/M
	"gtfs-l",    // L
	"gtfs-g",    // G
	"gtfs-jz",   // J/Z
	"gtfs-7",    // 7
	"gtfs-si",   // Staten Island Railway
}

// FeedPoller polls a single GTFS-RT feed and sends updates to the aggregator.
type FeedPoller struct {
	url        string
	name       string
	client     *http.Client
	aggregator *Aggregator
	interval   time.Duration
}

// NewFeedPoller creates a poller for the given feed suffix.
func NewFeedPoller(suffix string, aggregator *Aggregator, interval time.Duration) *FeedPoller {
	return &FeedPoller{
		url:  mtaBaseURL + suffix,
		name: suffix,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		aggregator: aggregator,
		interval:   interval,
	}
}

// Start begins polling in a goroutine. It blocks until ctx is cancelled.
func (fp *FeedPoller) Start(ctx context.Context) {
	log.Printf("feed/%s: starting poller (url=%s, interval=%s)", fp.name, fp.url, fp.interval)

	// Poll immediately on start.
	fp.poll()

	ticker := time.NewTicker(fp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("feed/%s: stopping poller", fp.name)
			return
		case <-ticker.C:
			fp.poll()
		}
	}
}

// poll fetches the feed once and sends updates to the aggregator.
func (fp *FeedPoller) poll() {
	resp, err := fp.client.Get(fp.url)
	if err != nil {
		log.Printf("feed/%s: HTTP error: %v", fp.name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("feed/%s: unexpected status %d", fp.name, resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("feed/%s: read error: %v", fp.name, err)
		return
	}

	feed := &gtfs.FeedMessage{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("feed/%s: protobuf panic recovered: %v", fp.name, r)
			}
		}()
		if err := proto.Unmarshal(body, feed); err != nil {
			log.Printf("feed/%s: protobuf decode error: %v", fp.name, err)
			feed = nil
		}
	}()
	if feed == nil {
		return
	}

	updates := extractUpdates(feed)
	if len(updates) > 0 {
		fp.aggregator.IngestFeed(fp.name, updates)
	}
	log.Printf("feed/%s: processed %d entities, %d train updates", fp.name, len(feed.GetEntity()), len(updates))
}

// extractUpdates extracts trainUpdate entries from a GTFS-RT FeedMessage.
// It pulls data from both TripUpdate (upcoming arrivals) and VehiclePosition entities.
func extractUpdates(feed *gtfs.FeedMessage) []trainUpdate {
	var updates []trainUpdate

	for _, entity := range feed.GetEntity() {
		// Only use VehiclePosition — shows actual current train locations.
		// TripUpdate has every future stop which floods the display.
		if vp := entity.GetVehicle(); vp != nil {
			routeID := vp.GetTrip().GetRouteId()
			route := NormalizeRoute(routeID)
			if route == "" {
				continue
			}

			stopID := vp.GetStopId()
			if stopID == "" {
				continue
			}

			status := MapVehicleStatus(vp.GetCurrentStatus())
			tripID := vp.GetTrip().GetTripId()

			updates = append(updates, trainUpdate{
				StopID: stopID,
				Route:  route,
				Status: status,
				TripID: tripID,
			})
		}
	}

	return updates
}

// inferStatus determines train status from a StopTimeUpdate.
// If the train is arriving within 30 seconds, it's INCOMING_AT.
// If it's at the stop (arrival time == 0 or past), it's STOPPED_AT.
// Otherwise it's IN_TRANSIT_TO.
func inferStatus(stu *gtfs.TripUpdate_StopTimeUpdate) pb.TrainStatus {
	now := time.Now().Unix()

	arrival := stu.GetArrival()
	if arrival != nil {
		arrTime := arrival.GetTime()
		if arrTime > 0 {
			diff := arrTime - now
			if diff <= 0 {
				return pb.TrainStatus_STOPPED_AT
			}
			if diff <= 30 {
				return pb.TrainStatus_INCOMING_AT
			}
			return pb.TrainStatus_IN_TRANSIT_TO
		}
	}

	return pb.TrainStatus_IN_TRANSIT_TO
}

// StartAllPollers launches all 9 feed pollers with staggered start times.
func StartAllPollers(ctx context.Context, aggregator *Aggregator, interval time.Duration) {
	for i, suffix := range feedSuffixes {
		poller := NewFeedPoller(suffix, aggregator, interval)
		stagger := time.Duration(i) * 1500 * time.Millisecond // 1.5s stagger

		go func(p *FeedPoller, delay time.Duration) {
			select {
			case <-time.After(delay):
				p.Start(ctx)
			case <-ctx.Done():
				return
			}
		}(poller, stagger)
	}
	fmt.Printf("started %d feed pollers\n", len(feedSuffixes))
}
