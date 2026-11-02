package settings

import "fmt"

// WeatherKey switches on the optional weather (wind, rain and snow) lookup for trips.
const WeatherKey = "CARTRAK_WEATHER_LOOKUP"

var weatherOptions = []Option{{"", "Off (nothing is sent)"}, {"on", "On: look up the wind for trips"}}

func init() {
	Defs = append(Defs, Def{Key: WeatherKey, Label: "Weather lookup for trips (optional)", Options: weatherOptions,
		Help: "When on, CarTrak asks a free weather service (Open-Meteo, no account) for the wind, rain and snow along each trip, so a trip can say whether it had a headwind, a tailwind, rain or snow, and what that cost. In the background it also fills in the last few months of trips, a few days at a time, so it can measure what rain costs your car. It sends only a map cell about 30 miles wide and the date, never your exact position, and remembers the answers. Off by default."})
	validators[WeatherKey] = func(v string) error {
		for _, o := range weatherOptions {
			if o.Value == v {
				return nil
			}
		}
		return fmt.Errorf("choose off or on")
	}
}

// WeatherLookup says whether the wind lookup is switched on.
func (s *Settings) WeatherLookup() bool { return s.Value(WeatherKey) == "on" }
