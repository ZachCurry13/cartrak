// Package tripwhy explains why a trip used more or less energy than usual: how the speed mix, the temperature and
// stopping compare with the driver's own history, and how much is left over (wind, hills, heating and cooling).
package tripwhy

import (
	"github.com/ZachCurry13/cartrak/internal/smooth"
)

// Reading is one stored reading during the trip.
type Reading struct {
	TS       int64
	Odo      float64 // km
	Kwh      float64 // energy left
	Speed    float64 // km/h
	OutsideC *float64
}

// Speed band history: kilometers per kWh for a speed range.
type Band struct {
	LoKph, HiKph float64
	KmPerKwh     float64
}

// Baseline is what the driver's history says.
type Baseline struct {
	Bands       []Band             // by speed
	Overall     float64            // km per kWh over all that driving
	TempKmPKwh  map[string]float64 // km per kWh by temperature group name
	TempOverall float64
}

// Factor is one reason. Pct is how much more (+) or less (-) energy it made the trip use.
type Factor struct {
	Key  string  `json:"key"`
	Pct  float64 `json:"pct"`
	Note string  `json:"note"`
}

// Facts are plain numbers about the trip, for the screen to word in the reader's units.
// Wind is the wind met on the trip (from the optional weather lookup).
type Wind struct {
	HeadKph   float64 `json:"headKph"` // positive is a headwind, negative a tailwind
	CrossKph  float64 `json:"crossKph"`
	GustKph   float64 `json:"gustKph"`
	EffectPct float64 `json:"effectPct"` // rough change in energy used because of it
}

type Facts struct {
	Km          float64  `json:"km"`
	AvgKph      float64  `json:"avgKph"` // while moving
	MaxKph      float64  `json:"maxKph"`
	Share100    float64  `json:"share100"` // share of the distance above 100 km/h
	Stops       int      `json:"stops"`
	StoppedPct  float64  `json:"stoppedPct"` // share of the trip's time spent stopped
	TempC       *float64 `json:"tempC"`
	TempGroup   string   `json:"tempGroup"`
	IdleKwh     float64  `json:"idleKwh"`     // energy used while stopped (climate control, systems)
	HardBrakes  int      `json:"hardBrakes"`  // big drops in speed between two readings
	QuickStarts int      `json:"quickStarts"` // big gains in speed between two readings
	Wind        *Wind    `json:"wind"`
	RegenKm     *float64 `json:"regenKm"`  // range the car says braking gave back during the trip (its own counter)
	RegenKwh    *float64 `json:"regenKwh"` // the energy that range is worth, at this trip's own use per kilometer
}

// Tag is one short reason shown next to a trip. Bad means it cost energy.
type Tag struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Bad   bool   `json:"bad"`   // cost this trip energy
	Level string `json:"level"` // "bad" (cost energy), "good" (saved energy) or "info" (describes the trip, no clear effect)
}

// Report is the explanation.
type Report struct {
	Ready         bool           `json:"ready"`
	KwhPerKm      float64        `json:"kwhPerKm"`
	UsualKwhPerKm float64        `json:"usualKwhPerKm"`
	VsUsualPct    float64        `json:"vsUsualPct"`
	Factors       []Factor       `json:"factors"`
	Tags          []Tag          `json:"tags"`
	Facts         Facts          `json:"facts"`
	Smooth        *smooth.Result `json:"smooth"` // how firmly the trip was driven, when the car's torque readings allow
}

const significant = 3.0 // percent: smaller effects are not worth a line

func bandIndex(b []Band, kph float64) int {
	for i := range b {
		if kph >= b[i].LoKph && kph < b[i].HiKph {
			return i
		}
	}
	return -1
}

// Build explains a trip. tempGroup is the name of the temperature group of the trip's average temperature.
func Build(rs []Reading, tripKwh float64, base Baseline, tempGroup string, regenKm *float64) Report {
	return BuildWithWind(rs, tripKwh, base, tempGroup, regenKm, nil)
}
