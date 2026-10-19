# Vehicle health

The **Health** tab shows the numbers behind a warning light before it comes on: the 12 V battery, the main battery, oil and fuel, tires, trouble codes and alerts. It is arranged for the kind of car you have.

Everything comes from what Ford sends. CarTrak shows only what your car reports, and each value says how old it is when it is more than two hours old. A parked car sends some values only when it wakes up (for example after a drive or a charge), so a battery reading can be a few days old. That is Ford and the car, not a CarTrak fault.

## What is shown

| Section | What it tells you | Cars |
|---|---|---|
| **Needs a look** | Everything below that is flagged, in one list | all |
| **Warnings and codes** | Stored trouble codes (with the system they belong to), Ford alerts, warning lights that are on | all |
| **12 V battery** | Voltage, charge percent and load status, with a daily trend. A weak 12 V battery causes many odd faults, even in an electric car | all |
| **Main battery** | Charge shown on the dash and the actual charge, energy left, estimated capacity, battery status, voltage, temperature, current, range, with a capacity trend | electric, hybrid, plug-in hybrid |
| **Engine and oil** | Oil life left, fuel level and range, coolant and oil temperature, hybrid mode, with an oil-life trend | gas, hybrid, plug-in hybrid |
| **Parked drain** | How much charge the car loses while parked: percent a day (and kWh) over 7 and 30 days, and the recent parked stays | electric, plug-in hybrid |
| **Tires** | Each tire's pressure against the sticker value, and the car's own tire status | all |
| **Car systems** | Automatic software updates, alarm, parking brake, car mode, remote start time left | all |

Flags: the 12 V battery is marked **Check** below 11.9 V or 50 percent charge, and **Act** below 11.6 V or 30 percent. A parked car often reads a little lower (about 12.0 V, 50 to 70 percent) until it next wakes and recharges the battery, so those readings get an explanation instead of a flag. Oil life is **Check** at 15 percent and **Act** at 5 percent. Fuel is **Check** at 10 percent. A main battery status other than Normal, a tire the car says is low, any trouble code and any warning light are flagged too.

**The 12 V percentage is the car's estimate, not a test.** If a shop has tested the battery and it is fine, an admin can tap **It was tested and is fine** under the 12 V battery. That quiets its warnings and its push alert for 90 days (the numbers still show), and **Resume warnings** turns them back on.

Honest limits:
- **Battery health is an estimate.** Ford does not send a state-of-health figure. The estimated capacity is the car's own number; CarTrak charts it so you can watch for a steady fall over months. It needs a couple of days of readings before a trend appears.
- **Trouble codes** appear only when Ford reports one. "None reported" means Ford reported none; it is not a full scan. Reading every code and live battery cell voltages needs an OBD-II adapter, which CarTrak does not use yet.
- **Read only.** The Ford API CarTrak uses cannot send commands (start, lock, charge schedules). It can read them.

## Gas, electric or hybrid
CarTrak decides which sections apply in this order:
1. **Your choice:** More, Settings, **Vehicle type**.
2. **Ford's vehicle record:** its `engineType` (for example BEV for a Mustang Mach-E).
3. **What the car reports**, as a fallback: a traction battery means electric, a fuel level means a gas engine, both together means a hybrid (with a plug-in status, a plug-in hybrid).

Why not simply look for oil life? Because a battery-electric car reports an oil life of 100 percent, a hybrid mode and an engine coolant temperature even though it has no engine. CarTrak ignores those unless the car also reports fuel.

The type also tidies the rest of the app: a gas or regular hybrid car has no Charging tab or Charging card, a battery car has no oil, and a gas car has no Battery card but a Fuel and oil card.

If it guesses wrong, choose the right type in Settings.

## Trying each type without a car
The built-in demo car can pretend to be each type. Set `CARTRAK_PROVIDER=demo` and `CARTRAK_DEMO_POWERTRAIN` to `ev` (default), `gas`, `hev` or `phev`.

## Parked drain
CarTrak looks at the stretches when the car sat for at least three hours, not driving and not charging, and divides the battery percent lost by the hours parked. Many electric cars lose about 1 percent a day or less. It marks **Check** at 2 percent a day or more and **Act** at 4 or more. Things that keep a car awake include Sentry or similar modes, scheduled preconditioning, a weak 12 V battery and frequent app wake-ups. Ford updates the battery level only when the car wakes, so the number is a guide, not a measurement.

## Data check
**More, Data check** looks through the last 90 days of what CarTrak recorded for things that seem wrong: a stretch with no odometer reading (usually the car sleeping, or CarTrak not running), an odometer that went backwards, trips that overlap, a trip that lasted over a day, a trip with an impossible average speed, and finished trips with almost no distance. It only reads; it never changes or deletes anything, and each trip finding links to that trip so you can judge it. For checking the odometer against your trips, see the check on the Mileage log.

## Battery health
**Health** has a Battery health card: the battery size as the car itself estimates it, month by month, and a plain verdict once there are at least two months of readings: steady (within about 3 percent), a gentle decline (up to about 8 percent), or worth watching. The estimate moves with temperature and how full the battery is, so a small winter dip is normal, and this is not a lab test.

## Fixing a bad trip
Admins see **Fix this trip** on a trip's page. **Delete this trip** removes a duplicate or junk trip (and any public link made from it); odometer and battery history are not touched. **Close this trip** finishes a trip that never ended, which needs 30 minutes without readings. A trip that is still in progress cannot be changed this way. The Data check page lists the trips worth a look.

## Debug report
When something looks wrong (a trip that is too long, an "unlocked" alert you know is wrong, a charge that does not add up), open **More, Debug report** soon afterwards, choose how far back it happened and press **Make report**. Copy it or download it and send it. It lists every reading in that time (ignition, speed, plug, charge state, power, battery, odometer, lock), the raw lock and door fields the car sent each time the lock state changed, the trips and charging sessions CarTrak made from it, and the alerts it sent. It leaves out where the car was, sign-ins, keys and the VIN, but read it before sending. Admins only.

## A road trip with the car left on
A car that is taking charge and not moving counts as parked even if the ignition was left on, so a road trip with a fast-charge stop is recorded as one trip before it and one after. Combine them into one journey on the Trips page if you want them together. Trips recorded before this change are not split automatically; use **Fix this trip** on a trip page to delete a bad one.
