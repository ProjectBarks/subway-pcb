package mta

import (
	"strings"

	pb "github.com/ProjectBarks/subway-pcb/service/gen/subwaypb"
	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

// routeNormMap normalizes express variants and shuttle alternate IDs
// to their canonical route strings.
var routeNormMap = map[string]string{
	"5X":  "5",
	"6X":  "6",
	"7X":  "7",
	"FX":  "F",
	"SF":  "FS",
	"SR":  "GS",
	"H":   "FS",
	"SIR": "SI",
}

// NormalizeRoute normalizes an MTA route_id string to its canonical route string.
// Express variants are collapsed (e.g. "5X" -> "5") and shuttle alternate IDs
// are mapped (e.g. "SF" -> "FS"). Returns empty string for empty input.
func NormalizeRoute(routeID string) string {
	r := strings.ToUpper(strings.TrimSpace(routeID))
	if r == "" {
		return ""
	}
	if mapped, ok := routeNormMap[r]; ok {
		return mapped
	}
	return r
}

// NormalizeStopID strips the trailing direction letter (N/S) from an MTA stop_id
// to produce the parent station ID. For example, "101N" -> "101", "A02S" -> "A02".
func NormalizeStopID(stopID string) string {
	if len(stopID) == 0 {
		return stopID
	}
	last := stopID[len(stopID)-1]
	if last == 'N' || last == 'S' {
		return stopID[:len(stopID)-1]
	}
	return stopID
}

// MapVehicleStatus converts a GTFS-RT VehicleStopStatus to our proto TrainStatus.
func MapVehicleStatus(vs gtfs.VehiclePosition_VehicleStopStatus) pb.TrainStatus {
	switch vs {
	case gtfs.VehiclePosition_STOPPED_AT:
		return pb.TrainStatus_STOPPED_AT
	case gtfs.VehiclePosition_INCOMING_AT:
		return pb.TrainStatus_INCOMING_AT
	case gtfs.VehiclePosition_IN_TRANSIT_TO:
		return pb.TrainStatus_IN_TRANSIT_TO
	default:
		return pb.TrainStatus_STATUS_NONE
	}
}
