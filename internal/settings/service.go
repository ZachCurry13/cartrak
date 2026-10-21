package settings

import (
	"fmt"
	"strconv"
)

// ServiceMinutesKey is how long the car must stay at a service place (a dealership, tire shop or repair shop) before CarTrak asks
// whether it was serviced.
const ServiceMinutesKey = "CARTRAK_SERVICE_MINUTES"

var serviceMinutes = []Option{{"", "20 minutes (default)"}, {"45", "45 minutes"}, {"60", "1 hour"}, {"120", "2 hours"}, {"off", "Never ask"}}

func init() {
	Defs = append(Defs, Def{Key: ServiceMinutesKey, Label: "Ask about a service visit after", Options: serviceMinutes,
		Help: "When the car stays this long at a place you marked as a dealership, tire shop or repair shop (More, Places), CarTrak asks after you leave whether it was serviced, what was done and what it cost."})
	validators[ServiceMinutesKey] = func(v string) error {
		for _, o := range serviceMinutes {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the times")
	}
}

// ServiceMinutes is the stay that counts as a service visit, or 0 when CarTrak should never ask.
func (s *Settings) ServiceMinutes() int {
	switch v := s.Value(ServiceMinutesKey); v {
	case "off":
		return 0
	case "":
		return 20
	default:
		n, _ := strconv.Atoi(v)
		return n
	}
}
