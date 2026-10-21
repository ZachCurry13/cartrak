# Home Assistant and other smart-home tools

CarTrak has a small read-only address that a smart-home tool can poll for the car's state: battery, range, plugged in, charging, locked and a few more values. It never includes the VIN or any location.

## 1. Make a device token
Open **More, Phone tracking** and add a device called "Home Assistant". Copy its token (it is shown once). The token can read this feed and can send phone location points; it cannot sign in or change anything. Remove the device to revoke it.

## 2. Add sensors
In Home Assistant, add this to `configuration.yaml` (use your own CarTrak address and token):

```yaml
rest:
  - resource: https://cartrak.example.com/api/ha/state
    scan_interval: 120
    headers:
      Authorization: Bearer YOUR_DEVICE_TOKEN
    sensor:
      - name: "Car battery"
        value_template: "{{ value_json.batteryPct }}"
        unit_of_measurement: "%"
        device_class: battery
      - name: "Car range"
        value_template: "{{ (value_json.rangeKm * 0.621371) | round(0) }}"
        unit_of_measurement: "mi"
      - name: "Car charging power"
        value_template: "{{ value_json.chargeKw }}"
        unit_of_measurement: "kW"
    binary_sensor:
      - name: "Car plugged in"
        value_template: "{{ value_json.plugged }}"
        device_class: plug
      - name: "Car charging"
        value_template: "{{ value_json.charging }}"
        device_class: battery_charging
      - name: "Car unlocked"
        value_template: "{{ value_json.locked == false }}"
        device_class: lock
```

Then, for example, an automation can turn on a charger plug, or send "plug in the car" when the car is below a level in the evening and `Car plugged in` is off.

## What you get
`GET /api/ha/state` with `Authorization: Bearer <token>` returns JSON with: `ready`, `ts` (when Ford reported it), `ageSeconds`, `batteryPct`, `rangeKm`, `energyKwh`, `capacityKwh`, `plugged`, `charging`, `chargeKw`, `locked`, `ignition`, `speedKph`, `odometerKm`, `battery12vVolts` and `outsideC`. Distances are in kilometers. Values Ford did not send are `null`. Ford updates some values only when the car wakes, so check `ageSeconds`.

If you protect CarTrak with Cloudflare Access, allow this path for Home Assistant (a service token or a bypass rule), or reach CarTrak from Home Assistant on your home network.
