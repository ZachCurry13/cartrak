package server

import (
	"fmt"
	"net/http"
)

// The app keeps the choice between miles and kilometres on each device, and sends it with every call in the X-Units
// header. Anything the server writes in words for a person (a message, a finding) uses it; numbers in the API itself stay
// in kilometres and the app converts them. Without the header the answer is in miles, the app's default.

const kmPerMile = 1.609344

func metricReq(r *http.Request) bool { return r != nil && r.Header.Get("X-Units") == "metric" }

// distText writes a distance for a person.
func distText(km float64, metric bool, decimals int) string {
	if metric {
		return fmt.Sprintf("%.*f km", decimals, km)
	}
	return fmt.Sprintf("%.*f mi", decimals, km/kmPerMile)
}

// speedText writes a speed for a person.
func speedText(kph float64, metric bool) string {
	if metric {
		return fmt.Sprintf("%.0f km/h", kph)
	}
	return fmt.Sprintf("%.0f mph", kph/kmPerMile)
}
