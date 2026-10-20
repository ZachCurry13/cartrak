# Places, trip review and maintenance

## Places
**More, Places** lists the places you have named, such as Home or the office. To add one, do it while the car is there (**Save this place** uses the car's last reported position), or open any trip and use **Name the start** or **Name the end**. Pick how close counts as being there (100 m to 600 m).

Named places do three things:
- **Trips read "Home → Office"** in the trip list and on the trip page, wherever a trip started or ended inside a place.
- **Alerts** when the car arrives at or leaves a place (turn on *Arrived at a named place* and *Left a named place* under More, Notifications). The first reading after CarTrak starts only sets the starting point, and the car must get about a quarter farther than the place's size before it counts as left, so a position that wobbles at the edge does not repeat.
- They are for you: the public share links and the mileage log do not use them (the mileage log names cities, as before).

## Trips to review
Trips that no rule matched and that you have not labelled show as **N trips to review** at the top of Trips. The review screen lists them with **Work** and **Personal** buttons (admins). Labelling a trip here is the same as choosing it on the trip page: your choice is never changed by rules.

## Maintenance
**More, Maintenance** keeps service reminders by distance, by time, or both, for example a tire rotation every 6,000 miles or 6 months, or a cabin air filter every 2 years. The preset buttons are only starting points: use the intervals in your car's service schedule.

Each reminder counts from when you add it, using the car's odometer then. It shows how much is left and turns **soon** (under about 300 miles or 30 days) and **due**. When it becomes due you get one alert (turn on *Service due* under More, Notifications); **Done** starts the next interval from the current odometer. Reminders by distance need CarTrak to have an odometer reading.

## Charging stopped early
*Charging stopped early* is an alert for a charge that stops while still plugged in at a lower level than your recent sessions ended at (below 70% and not within 3 points of where earlier sessions stopped). That can mean a loose cable or a charger fault. If you always charge to the same limit, that level is learned and never flagged. It replaces *Charging complete* for that session. When the car itself reports a **fault** or that it **lost contact with the charger**, the alert says so and does not wait for the level to be low. The session that has just ended is not counted as one of the earlier sessions.

## Slow fast charge and car-reported problems
*Slow fast charge* fires when a fast charge that started below 60% averaged under 70% of the median of your earlier fast charges (at least three of them, each at least ten minutes). *Car-reported problem* fires, at most once a day for each check, when the car reports something other than normal for the main battery, the tire pressure system or the 12 V load.

## Parked watch (experimental)
Ford no longer gives the Mach-E an alarm that reports to apps, but the car's motion sensors are in the data CarTrak receives. **More, Parked watch** is a trial of using them. A parked car reads close to zero; CarTrak checks every reading of a parked car and, for devices that turned on **Bump while parked**, alerts when the motion rises above the level set under More, Settings (Normal 1.0, Sensitive 0.6, Hard knocks only 2.0, or Off). **Car alarm** alerts when the car reports its alarm or panic alarm going off.

The honest limit: CarTrak only sees what the car reports, and a parked car reports now and then, not continuously. A knock lasts a fraction of a second, so it is only caught if a reading happens to land during or just after it. The Parked watch page shows how many parked readings were checked, the strongest motion seen, and every reading above a light touch, so after a week or two you can see whether it catches anything useful. **Start a fresh trial** clears the list. It does not replace a dash cam with parking mode, which records what happened; if you use one, the time of an alert tells you which clip to look at. If someone drives or tows the car away, the existing *Trip started* alert covers it.

## Charging prices at a place

Open a place under **More, Places** and set **Price per kWh here**: a rate for charging there that replaces your usual home or public price (enter 0 if it is free). Tick **Counts as home charging** to treat the place like home. Leave the price empty and charging there uses the usual prices. A place named Home counts as home without any setting. A charging session you set to Home or Away by hand still uses your choice. The usual prices, the cheaper night rate and the charging alerts are under **Charging, Charging settings**.
