# Fuel (gas and hybrid cars)

CarTrak was built around an electric car, but a gas or hybrid car gets its own fuel screens: how much each drive burned, what you
spend, and how far a gallon goes. It all starts from the **fuel gauge** the car reports, which is a percent of the tank, so CarTrak
first has to learn the size of the tank.

## Fill-ups and the tank size
When the gauge jumps up by 8 percentage points or more while the car is not moving, CarTrak keeps it as a **fill-up** and sends one
alert (**Filled up?**, if that alert is on). Open **More, Fuel** and enter, for that fill-up:

- the **gallons** you put in (liters, if your device uses kilometers), and
- the **price per gallon** (or per liter). You enter the price every time, because it changes from pump to pump.

From the gallons and how far the gauge climbed, CarTrak works out the **tank size**: 6 gallons that took the gauge from 30% to 70%
is a 15 gallon tank. It uses the middle of the sizes your fill-ups imply, so one odd gauge does not move it, and it counts only
fill-ups that climbed at least 15 points (a small top-up is too coarse to say). It gets better with every fill-up that starts well
below full. **Not a fill-up** dismisses a jump that was not one.

You can also **add a fill-up by hand** (date, gallons, price, and if you like the gauge before and after, which lets that fill-up
teach the tank size too).

## What it shows
- **Trips** show the tank at each end, and, once the tank size is known, the **fuel used, the economy (mpg, or L/100 km) and what the
  drive cost** at the price of your latest fill-ups. Until then a trip says how far a full tank would go at its pace. The gauge moves
  in coarse steps, so single drives say they are estimates, and a drive with a stop at the pump in the middle is left without a number
  rather than a wrong one.
- **Home** opens on the fuel tank (level, range and oil life) for a car that reports no battery level, and a line under **Needs
  attention** reminds you of fill-ups waiting for their gallons and price.
- **Fuel** (More) lists the fill-ups with what each cost and the economy since the one before (from the odometer and the gauge), and
  the month so far.
- **Insights, Fuel** shows what you spent by month and the economy by month. **Records** adds the best economy on a drive of 10 miles
  or more, the farthest between fill-ups, the biggest fill-up and the cheapest gas.
- **Wrapped** gets a **Fuel** slide in place of the energy ones, and its infographic shows the fuel bought and spent.
- Achievements about charging and battery efficiency are left out for a car that cannot plug in.

## Alerts
- **Low fuel**: the tank fell to 15% or less, once, until it has been filled. Off by default.
- **Filled up**: the gauge jumped, so a fill-up is waiting for its gallons and price. Off by default.

## Units
Amounts are kept in US gallons and the price per gallon. On a device set to kilometers (More, Display and data) everything reads in
liters and liters per 100 km, and the boxes take liters and a price per liter.

## Good to know
- A plug-in hybrid shows both: the Charging screens for the battery and Fuel for the tank.
- Older trips get their fuel too, because it comes from the readings CarTrak already keeps, as soon as the tank size is known.
- The debug report has a **FUEL** section (the tank size, the fill-ups and the gauge over time) to send when a number looks wrong.
