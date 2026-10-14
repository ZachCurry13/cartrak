# Road trips, charging and efficiency

## Energy left
The **Live** page has an **Energy left** tile in kWh (and about how much the battery holds when full), and Home shows the same in the Battery card. It is the car's own reading, so it changes only when Ford sends a new one, not every second.

## Miles until your reserve
If you do not like to go below a certain level on a road trip, set it once under **More, Display and data, Road-trip reserve (%)** (default 20, kept on this device). Home (Battery card) and Live then show how far you can still go before the battery reaches it, for example "Until 20%: 142 mi".

How it is worked out: the car's own range estimate, scaled to the part of the battery above your reserve (range x (charge - reserve) / charge). So it follows how the car expects to be driven right now. A hard climb, headwind or cold weather can use the battery faster than the estimate.

## Time until 80%, 100% or another percent
While the car is charging, the Charging card on Home and the **Time to target** tile on Live show how long until the battery reaches a target.
- On Home choose **80%**, **90%**, **100%** or **Other** and type any percent. On Live, tap the tile to switch between 80, 90 and 100. The choice is kept on this device.
- For **100%** it uses the car's own "time to full" when it reports one.
- Otherwise it is an **estimate** from the current charging power and the battery size. Home or other AC charging is steady, so this is usually close. Fast (DC) chargers slow down sharply above 80 percent, so for fast charging the estimate assumes about a third of the speed above 80 percent. Treat it as a guide, not a promise.
- It needs a reading while charging, so after you plug in it appears at the next update from the car.

## Trip efficiency (miles per kWh)
Each trip now shows **Energy used** and **Efficiency**: miles per kWh, or kWh per 100 km if your units are metric (More, Display and data). The trips list shows efficiency on each trip, and the period summary shows the total energy used and the overall figure.
- For new trips the energy used is the drop in the car's own **energy left** reading from the start to the end of the trip.
- For trips recorded before this was added, it is **estimated** from the battery percentage drop times the battery's estimated capacity, and shown with "about" or an approximate sign.
- A figure appears only when there is a real distance (over 1 mile) and a real amount of energy (over 0.3 kWh), and never when the battery gained charge (for example a long downhill).
- Ford updates some battery values only when the car wakes, so a very short trip can look odd. Longer trips are more reliable.
- While a trip is in progress, the **This trip** tile on Live shows its efficiency so far.

## Range between two battery levels
Home (under the battery) shows how far a stretch of battery goes, for example "20 to 80%: 142 mi from your last 30 days of driving". The two levels are your **road-trip reserve** (the low end) and your **charging target** (the high end, 80% unless you changed it on Live).

How it is worked out: from your own trips. CarTrak adds up the distance and the battery percent used over your finished trips in the last 30 days (trips of at least 2 km that used at least 1 percent; it needs a couple of trips and at least 8 percent in total), which gives the real distance per percent. That already includes the weather, your speeds and the car's hidden top and bottom buffer, so it needs no battery capacity. When there is too little driving it says it is using the car's own range figure instead.

The **Trips** page has a **Range by month** table with the same figure for each month, so you can see how it drops in winter and recovers in summer. It fills in as trips build up.

## Charging history and cost
The **Charging** tab lists each session with the energy that went into the battery, the battery levels, how long it took and the peak power. The energy is the difference in the car's own energy-left reading; for older sessions without it, it is estimated from the battery percent and marked "(estimate)".

To see what it costs, an admin sets **Price per kWh at home** (and, if you like, a price away from home and a currency symbol) in More, Settings. Then each session shows its cost and the top of the tab shows each month: sessions, kWh, cost, miles driven and **cost per mile**. On a slow (AC) charger some of the wall energy is lost as heat, so the cost is the energy added divided by the charging efficiency, times the price. Until CarTrak has measured your efficiency it uses the **Slow-charging efficiency** setting in More, Settings (90% by default). Fast (DC) sessions use the energy added.

**Measuring your real charging loss:** the car reports what went into the battery, and your charger reports what it delivered. On each session, admins can type the charger's kWh (from the charger's app or a meter) into **kWh from the charger**. The session then shows how much of it reached the battery, for example 87%. Once at least three slow-charging sessions have the number, CarTrak uses your measured average (weighted by energy) instead of the setting, and the Charging cost card says so. A number lower than the battery received is refused as a typing mistake, and clearing the box removes the entry.

**Home or away:** pick **Home** on one session you charged at home. CarTrak remembers that place and treats later sessions within about 150 m as home; everything else is priced as away from home. You can change any session by hand, and **Automatic** returns it to CarTrak's guess. Until you mark one, every session uses the home price.

## Road-trip planner
**More, Road-trip planner** tells you how many charging stops a trip needs. Enter the distance, your starting charge (it starts at the car's current charge) and the average charging power you get over a stop (many cars average well below their peak; 100 kW is a starting guess and is remembered on this device). It uses how far your battery goes per percent, from your own recent driving (or the car's range figure when there is too little driving), your road-trip reserve and your charge target from Display and data and Live.

It lists each stop: about how far into the trip, what charge you arrive with (your reserve), what to charge to (only as much as the rest of the trip needs, up to your target), and about how long it takes, then the charge you arrive with. It is a guide, not a route: hills, highway speed, wind and cold use more battery, and fast chargers slow down above 80%.

## Journeys: combining trips
A road trip or a work day with several stops is many trips. On **Trips**, admins can tap **Select trips to combine**, tap the trips that belong together (finished trips that are not already in a journey), name it and press **Combine**. The journey has its own page with:
- total distance, driving time and elapsed time;
- the energy used and the efficiency (estimated for trips recorded without the car's energy reading);
- the stops between legs, with how long the car was parked and how much it charged;
- what charging cost: the cost of the charging sessions during the journey when they have a price, otherwise an estimate from the energy used at your price per kWh (set in More, Settings);
- a map with every leg.

You can remove a trip from a journey, rename it, or **Ungroup trips**. The trips themselves are never deleted, and they keep counting in the mileage log and every total as before.

## Efficiency insights
**More, Efficiency insights** shows what changes your miles per kWh, from the last year of finished trips of at least 3 miles that have an energy reading:
- **By temperature:** below freezing, 0 to 10 °C, 10 to 20 °C and 20 °C and above, each with its efficiency and a bar.
- **By month:** efficiency and average outside temperature, so you can see the winter dip and summer recovery.
- **Most and least efficient trips** (needs at least six trips), each linking to the trip.

Cold weather, heating, highway speed and wind use more energy. Trips from before CarTrak recorded the car's energy reading use the battery percent and are less exact.
