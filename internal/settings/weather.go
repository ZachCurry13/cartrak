package settings

import "fmt"

// WeatherKey switches on the optional weather (wind, rain and snow) lookup for trips.
const WeatherKey = "CARTRAK_WEATHER_LOOKUP"

var weatherOptions = []Option{{"", "Off (nothing is sent)"}, {"on", "On: look up the wind for trips"}}

func init() {
	Defs = append(Defs, Def{Key: WeatherKey, Label: "Weather lookup for trips (optional)", Options: weatherOptions,
		Help: "On: CarTrak asks Open-Meteo (free, no account) for wind, rain and snow along each trip, so a trip can show what the weather cost. It also fills in recent trips in the background to learn what rain costs your car. Only a map cell about 30 miles wide and the date are sent. Off by default."})
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
