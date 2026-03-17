package mode

import (
	pb "github.com/ProjectBarks/subway-pcb/server/gen/subwaypb"
)

// TrackMode renders the standard subway map view — LEDs light up with route
// colors at stations where trains are currently stopped.
type TrackMode struct{}

func (m *TrackMode) Name() string        { return "track" }
func (m *TrackMode) Description() string { return "Live subway map — trains shown at their current stations" }

func (m *TrackMode) Render(ctx RenderContext) ([]byte, error) {
	trains := ctx.Aggregator.GetStationTrains()
	pixels := make([]byte, ctx.TotalLEDs*3)

	for i := 0; i < ctx.TotalLEDs; i++ {
		if i >= len(ctx.StationIDs) {
			break
		}
		sid := ctx.StationIDs[i]
		if sid == "" {
			continue
		}

		info, active := trains[sid]
		if !active {
			continue
		}

		// Look up color from theme
		routeKey := info.Route.String()
		color, ok := ctx.Theme.RouteColors[routeKey]
		if !ok {
			continue
		}

		pixels[i*3+0] = color[0]
		pixels[i*3+1] = color[1]
		pixels[i*3+2] = color[2]
	}

	return pixels, nil
}

// defaultRouteColors provides fallback colors matching the classic MTA scheme.
// Used when a theme doesn't define a color for a route.
var defaultRouteColors = map[pb.Route][3]uint8{
	pb.Route_ROUTE_1:  {255, 0, 0},
	pb.Route_ROUTE_2:  {255, 0, 0},
	pb.Route_ROUTE_3:  {255, 0, 0},
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
