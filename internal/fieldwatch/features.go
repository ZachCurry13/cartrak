package fieldwatch

// Features says what CarTrak uses a metric for, beyond showing it on the Car data page. A metric not listed here is
// either only shown there or not used at all, which is what the Field watcher page is for pointing out.
var Features = map[string]string{
	"speed": "Trips, Live, efficiency", "odometer": "Trips, mileage, drain", "ignitionStatus": "Trips, alerts",
	"heading": "Wind on trips", "outsideTemperature": "Efficiency by temperature, range",
	"xevBatteryStateOfCharge": "Home, Live, range", "xevBatteryActualStateOfCharge": "True battery level",
	"xevBatteryEnergyRemaining": "Energy per trip, battery size", "xevBatteryCapacity": "Battery health",
	"xevBatteryRange": "Home, planner", "xevBatteryMaximumRange": "True battery level",
	"xevBatteryTemperature": "Live", "xevBatteryVoltage": "Charging power, Live", "xevBatteryIoCurrent": "Charging power, parked energy",
	"xevTractionMotorVoltage": "Live", "xevTractionMotorCurrent": "Live", "engineCoolantTemp": "Live",
	"xevBatteryTimeToFullCharge": "Charging", "xevBatteryChargerEnergyOutput": "Charging",
	"xevBatteryChargeDisplayStatus": "Charging, alerts", "xevPlugChargerStatus": "Charging, alerts",
	"xevChargeStationPowerType": "Level 2 or fast", "xevBatteryPerformanceStatus": "Car-reported problem alert",
	"tirePressure": "Tire trends", "tirePressureStatus": "Low tire alert", "tirePressureSystemStatus": "Car-reported problem alert",
	"batteryVoltage": "12 V health", "batteryStateOfCharge": "12 V health", "batteryLoadStatus": "Car-reported problem alert",
	"doorLockStatus": "Unlocked alert", "doorStatus": "Left open alert", "windowStatus": "Left open alert", "hoodStatus": "Left open alert",
	"alarmStatus": "Parked watch", "panicAlarmStatus": "Parked watch", "acceleration": "Parked watch",
	"torqueAtTransmission": "Driving firmness", "brakeTorque": "Driving firmness",
	"tripXevBatteryRangeRegenerated": "Regen on trips", "remoteStartCountdownTimer": "Remote start chip",
	"xevChargeStationCommunicationStatus": "Charging",
}
