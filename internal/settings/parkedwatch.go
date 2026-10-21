package settings

import (
	"fmt"
	"strconv"
)

// Parked watch: how big a jolt on a parked car's motion sensors counts as a bump (meters per second squared).
const BumpKey = "CARTRAK_BUMP_THRESHOLD"

var bumpOptions = []Option{{"", "Normal (1.0)"}, {"0.6", "Sensitive (0.6)"}, {"2.0", "Hard knocks only (2.0)"}, {"off", "Off"}}

func init() {
	Defs = append(Defs, Def{Key: BumpKey, Label: "Parked watch: bump level", Options: bumpOptions,
		Help: "A parked car's motion sensors read close to zero. A reading above this level while the car is parked counts as a bump, for devices that turned on the Bump while parked alert. The car only reports now and then, so a short knock can be missed. See More, Parked watch."})
	validators[BumpKey] = func(v string) error {
		for _, o := range bumpOptions {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose one of the levels")
	}
}

// BumpThreshold is the motion level that counts as a bump, or 0 when the watch is off.
func (s *Settings) BumpThreshold() float64 {
	switch v := s.Value(BumpKey); v {
	case "":
		return 1.0
	case "off":
		return 0
	default:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	}
}
