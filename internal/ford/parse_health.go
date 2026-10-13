package ford

// parseHealth reads the values the Vehicle health page trends over time. A car that does not
// report one simply leaves it nil.
func parseHealth(f map[string]any, s *State) {
	s.FuelPct = numAt(f, "fuelLevel")
	s.OilLifePct = numAt(f, "oilLifeRemaining")
	s.HVVolts = numAt(f, "xevBatteryVoltage")
	s.CapacityKwh = numAt(f, "xevBatteryCapacity")
}
