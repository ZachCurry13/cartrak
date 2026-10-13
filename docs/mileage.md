# Mileage log

**More, Mileage log** (admins) turns your trips into a day-by-day business mileage log you can copy and paste into a sheet. It matches this layout:

| Date | Origin City | Destination City | Description | Total Miles Driven |
|---|---|---|---|---|
| 2026-09-15 | Springfield | Rivertown, Port Lakeside, Oakdale | Customer Visits | 82 |

and a **Total Mileage Recorded** line underneath.

## How to use it
1. Open More, **Mileage log**.
2. Type the **calendar weeks**, for example `26-28` (or one week, `26`), and the year. Or leave weeks empty and choose a from and to date.
3. Check the **Description** (default "Customer Visits") and which trips: **work trips only** (default), all, or personal.
4. Press **Build the log**. Look at the table, then press **Copy rows to paste into your sheet** (just the rows, to paste under your own headings) or **Copy with headings and total**. The copy is tab-separated, so each value lands in its own cell.

## How each row is made
- One row per **day**, by the date each trip started, in the time zone where that trip started.
- **Origin City:** the city where the day's first trip started.
- **Destination City:** the city where the car was parked at the end of each trip, in the order you visited them, **each city once** (no repeats). Your origin city is left out, because ending the day back home is not a destination.
- **Total Miles Driven:** the day's trip distances added up, in miles, rounded to a whole number. The total at the bottom is the sum of the rows shown.
- Only finished trips count. Trips still in progress are skipped.
- Calendar weeks are ISO weeks: Monday to Sunday, week 1 is the week that holds January 4th. Weeks 26-28 of 2026 are June 22 to July 12.

CarTrak only has trips from the day it was linked to your car, so weeks before that come back empty. Label trips as **work** first (More, Trip rules, or by hand on each trip) or choose "All trips".

## Where the city names come from
Your trips store coordinates, not city names. To name the city, your CarTrak server asks **OpenStreetMap's Nominatim** service:
- Only when an admin builds a log, never in the background.
- Each location is **rounded to about 100 m** before it is sent, and nothing else is sent: no name, account or trip details.
- Every answer is **remembered** in the database, so a place (home, a regular customer) is looked up once and a repeat run asks nobody.
- Lookups are spaced at least a second apart, as the service asks. A run looks up at most 40 new places; if more are needed the page says so and **Run again to finish** continues where it stopped.
- If you would rather not send locations anywhere, set **More, Settings, City lookup for the mileage report** to Off. Cities then appear only for places already remembered.
- You can point it at your own Nominatim server with **City lookup service (advanced)**, or with `CARTRAK_GEOCODE_URL`.

Note that parking locations, including your home if a trip ends there, are sent to that service when you use it.

## Settings
| Setting | Environment variable | Meaning |
|---|---|---|
| City lookup for the mileage report | `CARTRAK_GEOCODE_ENABLED` | Empty is on; `off` never sends locations |
| City lookup service (advanced) | `CARTRAK_GEOCODE_URL` | Another Nominatim-compatible address (https, or http on localhost) |

## Good to know
- The place name is the city, town or village Nominatim reports. A spot that is inside a township with no town name may be named by its county.
- Names come from OpenStreetMap, so a few places may be spelled differently from your own list (for example "Pt. Lakeside" versus "Port Lakeside"). Edit the pasted text in your sheet if you need a particular spelling.
- Trips with no end location (the car had no GPS fix) are listed in a note, and their city is missing from the row.
