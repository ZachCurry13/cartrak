package settings

import "strings"

// AFDCKeyKey is the free key for the US government's charger list (other fast-charging networks such as Electrify America).
const AFDCKeyKey = "CARTRAK_AFDC_KEY"

func init() {
	Defs = append(Defs, Def{Key: AFDCKeyKey, Label: "Charger list key (Electrify America)", Secret: true,
		Help: "Optional. Lets the long-trip planner add Electrify America chargers. Get a free key at developer.nlr.gov/signup and paste it here; it stays on the server. Without it only Superchargers are offered."})
}

// AFDCKey is the charger list key, "" when not set.
func (s *Settings) AFDCKey() string { return strings.TrimSpace(s.Value(AFDCKeyKey)) }
