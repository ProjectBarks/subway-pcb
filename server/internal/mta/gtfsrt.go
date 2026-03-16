package mta

import (
	"strings"

	pb "github.com/ProjectBarks/subway-pcb/server/gen/subwaypb"
	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
)

// routeMap maps MTA route_id strings to our proto Route enum values.
var routeMap = map[string]pb.Route{
	"1":  pb.Route_ROUTE_1,
	"2":  pb.Route_ROUTE_2,
	"3":  pb.Route_ROUTE_3,
	"4":  pb.Route_ROUTE_4,
	"5":  pb.Route_ROUTE_5,
	"5X": pb.Route_ROUTE_5, // Express variant
	"6":  pb.Route_ROUTE_6,
	"6X": pb.Route_ROUTE_6, // Express variant
	"7":  pb.Route_ROUTE_7,
	"7X": pb.Route_ROUTE_7, // Express variant
	"A":  pb.Route_ROUTE_A,
	"B":  pb.Route_ROUTE_B,
	"C":  pb.Route_ROUTE_C,
	"D":  pb.Route_ROUTE_D,
	"E":  pb.Route_ROUTE_E,
	"F":  pb.Route_ROUTE_F,
	"FX": pb.Route_ROUTE_F, // Express variant
	"G":  pb.Route_ROUTE_G,
	"J":  pb.Route_ROUTE_J,
	"L":  pb.Route_ROUTE_L,
	"M":  pb.Route_ROUTE_M,
	"N":  pb.Route_ROUTE_N,
	"Q":  pb.Route_ROUTE_Q,
	"R":  pb.Route_ROUTE_R,
	"W":  pb.Route_ROUTE_W,
	"Z":  pb.Route_ROUTE_Z,
	"S":  pb.Route_ROUTE_S,
	"SF": pb.Route_ROUTE_FS, // Franklin Ave Shuttle
	"SR": pb.Route_ROUTE_GS, // Grand Central Shuttle (Rockaway Park)
	"FS": pb.Route_ROUTE_FS,
	"GS": pb.Route_ROUTE_GS,
	"H":  pb.Route_ROUTE_FS, // Alternate ID for Franklin shuttle
	"SI": pb.Route_ROUTE_SI,
	"SIR": pb.Route_ROUTE_SI,
}

// MapRoute converts an MTA route_id string to the proto Route enum.
// Returns ROUTE_UNKNOWN for unrecognized routes.
func MapRoute(routeID string) pb.Route {
	r, ok := routeMap[strings.ToUpper(strings.TrimSpace(routeID))]
	if !ok {
		return pb.Route_ROUTE_UNKNOWN
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
