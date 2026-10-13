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
