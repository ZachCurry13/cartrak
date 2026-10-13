package cardata

import "github.com/ZachCurry13/cartrak/internal/ford"

// battery is the Battery and charging tab (and the fuel tank for cars with an engine).
func (s src) battery() Tab {
	var main, charge, engine *Group
	if s.pt.HasBattery() {
		main = group("Main battery", "The large battery that drives the car.",
			s.num("Charge shown on the dash", "pct", "The percentage on the car's display.", "xevBatteryStateOfCharge"),
			s.num("Actual charge", "pct", "The true charge of the battery cells. It includes a small buffer at the top and bottom that the car hides from the dash to protect the battery, so the two percentages differ by a few points. Plan with the dash number.", "xevBatteryActualStateOfCharge"),
			s.num("Energy left", "kwh", "Usable energy left, like gallons in a fuel tank. A kilowatt-hour (kWh) is the unit electricity is sold in.", "xevBatteryEnergyRemaining"),
			s.num("Battery size (estimated)", "kwh", "How much the battery can hold when full, as the car estimates it. It shrinks very slowly as the battery ages.", "xevBatteryCapacity"),
			s.num("Range now", "km", "How far the car expects to go on the current charge.", "xevBatteryRange"),
			s.num("Range when full", "km", "How far it expects to go on a full charge.", "xevBatteryMaximumRange"),
			s.text("Battery status", "The car's own check of the battery. Normal is what you want.", "xevBatteryPerformanceStatus"),
			s.num("Voltage", "volts", "The electrical pressure of the big battery. It moves a little with charge and is not a problem sign by itself.", "xevBatteryVoltage"),
			s.num("Temperature", "tempC", "Battery temperature. A cold battery charges slower and drives less far.", "xevBatteryTemperature"),
			s.num("Current", "amps", "Electricity flowing into the battery (charging) or out of it (driving). Close to zero when parked.", "xevBatteryIoCurrent"))
		charge = group("Charging", "What the charging cable and charger are doing.",
			s.text("Plug", "Whether a charging cable is connected to the car.", "xevPlugChargerStatus"),
			s.text("Charging status", "What the car says charging is doing: not ready, charging, complete and so on.", "xevBatteryChargeDisplayStatus"),
			s.text("Charging station", "Whether a charger is talking to the car. Not detected means no charger is connected.", "xevChargeStationCommunicationStatus"),
			s.text("Station type", "AC basic is an ordinary home or level 2 charger. DC means a fast charger.", "xevChargeStationPowerType"),
			s.num("Time to full", "minutes", "The car's estimate to reach 100 percent at the current charging speed. Only meaningful while charging.", "xevBatteryTimeToFullCharge"),
			s.num("Charger voltage", "volts", "The electrical pressure coming from the charger.", "xevBatteryChargerVoltageOutput"),
			s.num("Charger current", "amps", "How much electricity the charger is delivering.", "xevBatteryChargerCurrentOutput"),
			s.num("Energy delivered", "kwh", "How much energy the charger has put into the battery in this charge.", "xevBatteryChargerEnergyOutput"),
			s.num("Drive motor voltage", "volts", "Electricity going to the drive motor. Near zero when stopped.", "xevTractionMotorVoltage"),
			s.num("Drive motor current", "amps", "How much electricity the drive motor is using. Zero when stopped.", "xevTractionMotorCurrent"))
	}
	// A battery car also reports an oil life of 100 and an engine temperature, so those only show when there is a fuel tank.
	if s.pt.HasEngine() && (s.pt != ford.Unknown || ford.Number(s.f, "fuelLevel") != nil) {
		engine = group("Fuel and engine", "",
			s.num("Fuel level", "pct", "How full the fuel tank is.", "fuelLevel"),
			s.num("Fuel range", "km", "How far the car expects to go on the fuel left.", "fuelRange"),
			s.num("Oil life left", "pct", "How much of the engine oil's life is left. Plan an oil change when it gets low.", "oilLifeRemaining"),
			s.num("Coolant temperature", "tempC", "The temperature of the engine coolant.", "engineCoolantTemp"),
			s.num("Oil temperature", "tempC", "The temperature of the engine oil.", "engineOilTemp"),
			s.num("Engine speed", "rpm", "How fast the engine is turning, in revolutions per minute. Zero when it is off.", "engineSpeed"))
	}
	small := group("12 V battery", "The small battery, like in any car. It runs the lights, screens and computers and wakes the car up.",
		s.num("Voltage", "volts", "Around 12.4 to 12.8 V when the car is resting is healthy. Lower can mean it needs charging or testing.", "batteryVoltage"),
		s.num("Charge", "pct", "How full the small battery is.", "batteryStateOfCharge"),
		s.text("Load", "Whether the electronics are drawing a normal amount of power.", "batteryLoadStatus"))
	return tab("battery", "Battery and charging", "Everything about the batteries and charging.", main, small, charge, engine)
}
