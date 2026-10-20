package report

const kmPerMile = 1.609344

// Column is something a report can show for each trip.
type Column struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`   // the default heading
	Default bool    `json:"default"` // in the standard report
	Width   float64 `json:"-"`
	Number  bool    `json:"-"`
}

// Columns lists every column, in the order the standard report shows them.
var Columns = []Column{
	{Key: "value", Label: "Value", Default: true, Width: 12, Number: true},
	{Key: "distance", Label: "Miles", Default: true, Width: 10, Number: true},
	{Key: "rate", Label: "Mileage Rate", Default: true, Width: 13, Number: true},
	{Key: "from", Label: "From", Default: true, Width: 28},
	{Key: "to", Label: "To", Default: true, Width: 28},
	{Key: "date", Label: "Date", Default: true, Width: 12},
	{Key: "purpose", Label: "Purpose", Default: true, Width: 12},
	{Key: "track", Label: "Track Method", Default: true, Width: 14},
	{Key: "notes", Label: "Notes", Default: true, Width: 24},
	{Key: "vehicle", Label: "Vehicle", Default: true, Width: 16},
	{Key: "started", Label: "Time Started", Default: true, Width: 13},
	{Key: "ended", Label: "Time Ended", Default: true, Width: 13},
	{Key: "hours", Label: "Duration (hours)", Default: true, Width: 16, Number: true},
	{Key: "zone", Label: "Time Zone", Default: true, Width: 22},
	{Key: "weekday", Label: "Day of Week", Width: 12},
	{Key: "energy", Label: "Energy Used (kWh)", Width: 17, Number: true},
	{Key: "efficiency", Label: "Efficiency", Width: 14},
	{Key: "energycost", Label: "Energy Cost", Width: 13, Number: true},
	{Key: "costper", Label: "Cost per Mile", Width: 14, Number: true},
	{Key: "mpge", Label: "MPGe", Width: 9, Number: true},
	{Key: "avgspeed", Label: "Average Speed", Width: 14, Number: true},
	{Key: "socstart", Label: "Battery Start (%)", Width: 16, Number: true},
	{Key: "socend", Label: "Battery End (%)", Width: 16, Number: true},
	{Key: "odostart", Label: "Odometer Start", Width: 15, Number: true},
	{Key: "odoend", Label: "Odometer End", Width: 15, Number: true},
	{Key: "topspeed", Label: "Top Speed", Width: 11, Number: true},
	{Key: "id", Label: "Trip ID", Width: 8, Number: true},
}

var byKey = func() map[string]Column {
	m := map[string]Column{}
	for _, c := range Columns {
		m[c.Key] = c
	}
	return m
}()

// ValidKey says whether a column key exists.
func ValidKey(k string) bool { _, ok := byKey[k]; return ok }
