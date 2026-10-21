# Mileage log

**More, Mileage log** (admins) turns your trips into a day-by-day business mileage log you can copy and paste into a sheet. It matches this layout:

| Date | Origin City | Destination City | Description | Total Miles Driven |
|---|---|---|---|---|
| 2026-09-15 | Springfield | Rivertown, Port Lakeside, Oakdale | Customer Visits | 82 |

and a **Total Mileage Recorded** line underneath.

## How to use it
0. For the usual periods, tap **This week**, **Last week**, **This month** or **Last month** under *Which period*; the log is built straight away. When the period covers more than one calendar week, a line under the table gives the miles for each calendar week.
1. Open More, **Mileage log**.
2. Type the **calendar weeks**, for example `26-28` (or one week, `26`), and the year. Or leave weeks empty and choose a from and to date.
3. Check the **Description** (default "Customer Visits") and which trips: **work trips only** (default), all, or personal.
4. Press **Build the log**. Look at the table, then press **Copy rows to paste into your sheet** (just the rows, to paste under your own headings) or **Copy with headings and total**. The copy is tab-separated, so each value lands in its own cell.

## How each row is made
- **Miles:** each trip is counted in whole miles, **rounded up to the next mile** (0.1 mi counts as 1, 26.2 as 27), then the trips of the day are added. Under **Miles** you can choose to add the day first and round to the nearest mile instead. The choice is remembered on this device.
- One row per **day**, by the date each trip started, in the time zone where that trip started.
- **Origin City:** the city where the day's first trip started.
- **Destination City:** the city where the car was parked at the end of each trip, in the order you visited them, **each city once** (no repeats). Your origin city is left out, because ending the day back home is not a destination.
- **Total Miles Driven:** the day's trip distances added up, in miles, rounded to a whole number. The total at the bottom is the sum of the rows shown.
- Only finished trips count. Trips still in progress are skipped.
- Calendar weeks are ISO weeks: Monday to Sunday, week 1 is the week that holds January 4th. Weeks 26-28 of 2026 are June 22 to July 12.

CarTrak only has trips from the day it was linked to your car, so weeks before that come back empty. Label trips as **work** first (More, Trip rules, or by hand on each trip) or choose "All trips".

## Where the city names come from
Your trips store coordinates, not city names. To name the city, your CarTrak server asks **OpenStreetMap's Nominatim** service:
- When an admin builds a log, and in the background, a few places at a time, for the towns shown on Trips, the Day view and History.
- Each location is **rounded to about 100 m** before it is sent, and nothing else is sent: no name, account or trip details.
- Every answer is **remembered** in the database, so a place (home, a regular customer) is looked up once and a repeat run asks nobody.
- Lookups are spaced at least a second apart, as the service asks. A run looks up at most 40 new places; if more are needed the page says so and **Run again to finish** continues where it stopped.
- If you would rather not send locations anywhere, set **More, Settings, City names for trips and reports** to Off. Cities then appear only for places already remembered.
- You can point it at your own Nominatim server with **City lookup service** (under Advanced), or with `CARTRAK_GEOCODE_URL`.

Note that parking locations, including your home if a trip ends there, are sent to that service when you use it.

## Settings
| Setting | Environment variable | Meaning |
|---|---|---|
| City names for trips and reports | `CARTRAK_GEOCODE_ENABLED` | Empty is on; `off` never sends locations |
| City lookup service (advanced) | `CARTRAK_GEOCODE_URL` | Another Nominatim-compatible address (https, or http on localhost) |

## Good to know
- The place name is the city, town or village Nominatim reports. A spot that is inside a township with no town name may be named by its county.
- Names come from OpenStreetMap, so a few places may be spelled differently from your own list (for example "Pt. Lakeside" versus "Port Lakeside"). Edit the pasted text in your sheet if you need a particular spelling.
- Trips with no end location (the car had no GPS fix) are listed in a note, and their city is missing from the row.

## Does it add up? (the mileage check)
Under the log, a card compares the car's **odometer** with the trips CarTrak recorded for the same period. The odometer only moves when the car does, so the miles it moved are exact even if CarTrak missed a trip. The card says either **Matches** or how many miles are missing, and lists each stretch of driving that no trip covers, with its time window and the cities where the car was at each end (when they are already remembered).

Each stretch has a one-tap fix:
- **A whole missed trip:** **Add as work trip** or **Add as personal trip** creates the trip with the miles from the odometer, the start and end places and times, and the battery levels. It then counts in the log and in every other screen. Adding the same miles twice is refused.
- **Miles at the edge of a recorded trip:** when the stretch sits right before or after one trip (CarTrak noticed that trip a little late or closed it a little late), **Add to that trip** widens that trip instead of creating a new one.

Notes:
- The check counts work and personal trips together, because the odometer does not know the difference.
- Ford reports the odometer only when the car wakes, so the times inside a stretch are approximate. The total is exact.
- It needs at least two odometer readings in the period. Readings from before CarTrak was linked do not exist.
- Only admins can see and use it.

## When a drive's distance is wrong

The car's odometer sometimes updates late, so a drive can be recorded shorter than it was. Open the trip and use **Does the distance look wrong?**: **Check the road distance** asks the routing service for the length of a road between the positions the car reported, and **Use** puts that length on the trip; or type the real distance; or **Go back to the odometer**. The trip is marked as corrected, its road is drawn, and the mileage log, reports and efficiency use the corrected distance. A trip that the road estimate could not do because the road was a very different length shows a **Use** button for it right in that notice.

When the distance is **drastically** short (the start and end alone are much farther apart in a straight line than the odometer says), the trip page puts **This distance looks too short** near the top and asks straight away for two lengths: the road through the positions the car reported, and the drive straight from the start to the end, as a navigation app would plan it. Pick one, or type the real distance if you know it (when CarTrak noticed the trip late, both can still be short). Home lists such trips from the last 30 days under **Needs attention**. Nothing changes until you choose.
