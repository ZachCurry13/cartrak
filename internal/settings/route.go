package settings

import "fmt"

// RouteKey switches on the optional road-route estimate for trips that have only a few recorded positions.
const RouteKey = "CARTRAK_ROUTE_LOOKUP"

// RouteURLKey is the routing service to ask (an OSRM-compatible server). Empty uses the public demo server.
const RouteURLKey = "CARTRAK_ROUTE_URL"

var routeOptions = []Option{{"", "Off (nothing is sent)"}, {"on", "On: estimate the road for car-only trips"}}

func init() {
	Defs = append(Defs, Def{Key: RouteKey, Label: "Road estimate for trips and the route planner (optional)", Options: routeOptions,
		Help: "A trip with only a few positions is drawn as a straight line. On: CarTrak asks a routing service for the road between them and draws it dashed, as an estimate. The route planner uses it too, for real road distances (it sends those positions to the service). Off by default."})
	Defs = append(Defs, Def{Key: RouteURLKey, Label: "Routing service address (advanced)", Placeholder: "leave empty for the public OSRM server",
		Help: "An OSRM-compatible server. The public demo is for light use; your own keeps positions off the internet."})
	validators[RouteKey] = func(v string) error {
		for _, o := range routeOptions {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose off or on")
	}
}

// RouteLookup says whether the road estimate is switched on.
func (s *Settings) RouteLookup() bool { return s.Value(RouteKey) == "on" }

// RouteURL is the routing service address, or "" for the default.
func (s *Settings) RouteURL() string { return s.Value(RouteURLKey) }
