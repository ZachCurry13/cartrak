package settings

// chargingKeys are the settings about charging: what it costs, when the price is lower, how efficient a home charger is and
// the alerts about charging. The app shows them on their own page under Charging, so the owner of each car can find and
// change them without wading through the technical settings.
var chargingKeys = map[string]bool{
	PriceHomeKey: true, PricePublicKey: true, CurrencyKey: true, OffPeakPriceKey: true, OffPeakWindowKey: true,
	ChargeEffKey: true, ChargeBudgetKey: true, FuelUnitsKey: true, PetrolPriceKey: true, PetrolEconomyKey: true, GridCO2Key: true, LowBatteryKey: true, PlugReminderTimeKey: true, PlugReminderBelowKey: true,
}

// GroupOf is which page a setting is shown on: "charging", "remote" (its own page), or "" for the general settings.
func GroupOf(d Def) string {
	if d.Group != "" {
		return d.Group
	}
	if chargingKeys[d.Key] {
		return "charging"
	}
	return ""
}
