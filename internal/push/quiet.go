package push

// quiet are the everyday alerts that arrive without a sound or a buzz. They are the ones that fire many times a day (a trip
// started, a trip ended, plugged in), so with a sound each they felt random. They still show in the notification shade;
// warnings and anything that needs action (a low tire, a car alarm, a fault, a low battery) keep their sound.
var quiet = map[string]bool{
	TripStart: true, TripEnd: true, PlugIn: true, PlugOut: true, ArrivePlace: true, LeavePlace: true,
	Achievement: true, MonthlyRecap: true, WeeklySummary: true,
}

// Quiet says whether an alert kind is delivered silently.
func Quiet(kind string) bool { return quiet[kind] }
