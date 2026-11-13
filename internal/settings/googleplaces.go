package settings

import "strings"

// GooglePlacesKeyKey is a Google Places API key, for finding businesses by name in a pasted company list.
const GooglePlacesKeyKey = "CARTRAK_GOOGLE_PLACES_KEY"

func init() {
	Defs = append(Defs, Def{Key: GooglePlacesKeyKey, Label: "Google Places key (companies and free chargers)", Secret: true,
		Help: "Optional. Lets a pasted company list find businesses the way Google does, and shows how many plugs are free at chargers (Check now on the Charging page and at planned charging stops; each check is one billed lookup, in Google's pricier tier). Make a key in Google Cloud (Places API (New)), restrict it to that API and set a small budget alert; a few lookups a week cost nothing within Google's free allowance. Only the company name and Home's rough area are sent. It stays on the server. Without it the map's own search is used, which does not list many small businesses."})
}

// GooglePlacesKey is the key, "" when not set.
func (s *Settings) GooglePlacesKey() string { return strings.TrimSpace(s.Value(GooglePlacesKeyKey)) }
