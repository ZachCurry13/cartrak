package push

// The alerts that come from the calendar and the weather service.
const (
	WeatherWarning = "weather_warning"
	DayAhead       = "day_ahead"
)

func init() {
	Kinds = append(Kinds,
		Kind{WeatherWarning, "Weather warning", "The National Weather Service issued a warning for the roads (winter storm, ice, flood, fog, high wind) where the car is. Needs Weather warnings switched on under More, Integrations."},
		Kind{DayAhead, "Tomorrow's appointments", "At 7 pm, when your calendar has appointments tomorrow: the first one, the weather and warnings at them, and how much battery the day needs. Needs the calendar link under More, Integrations."})
}
