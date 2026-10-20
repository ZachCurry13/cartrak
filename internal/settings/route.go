package settings

import "fmt"

// RouteKey switches on the optional road-route estimate for trips that have only a few recorded positions.
const RouteKey = "CARTRAK_ROUTE_LOOKUP"

// RouteURLKey is the routing service to ask (an OSRM-compatible server). Empty uses the public demo server.
const RouteURLKey = "CARTRAK_ROUTE_URL"

var routeOptions = []Option{{"", "Off (nothing is sent)"}, {"on", "On: estimate the road for car-only trips"}}

func init() {
	Defs = append(Defs, Def{Key: RouteKey, Label: "Road estimate for trips without phone GPS (optional)", Options: routeOptions,
		Help: "A trip the car recorded with only a few positions is drawn as a straight line. When on, CarTrak asks a routing service for the road between those positions and draws it dashed, as an estimate (it is not where the car really went). It sends the start, the end and a few positions along the way to that service. Off by default."})
	Defs = append(Defs, Def{Key: RouteURLKey, Label: "Routing service address (advanced)", Placeholder: "leave empty for the public OSRM server",
		Help: "An OSRM-compatible server. The public demo server is for light use; you can run your own and put its address here, which also keeps your positions off the internet."})
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
