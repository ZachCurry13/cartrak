# Live view, road trips and data saver

## Live view
**Live** (the middle tab) is a full-screen dashboard of big tiles for charge, range,
speed, charging power, this trip, outside temperature and odometer. It is meant for
a tablet on a car mount, a laptop, or a phone you glance at. It keeps the screen
awake (where the browser allows it) and has a **Full screen** button.

New readings are **pushed** to the screen the moment CarTrak gets them from Ford
(server-sent events), so nothing polls and nothing is wasted. It works through a
Cloudflare tunnel.

## Road trip mode
Press **Road trip mode** on the Live page (admins). While you are driving, CarTrak
asks Ford for a reading every `CARTRAK_POLL_LIVE_SEC` seconds (default 30, never below 30)
instead of every 30. It turns itself off after 12 hours so it cannot be left on by accident.
Parked and charging cars are not polled faster.

How "live" it can really be depends on Ford: the car's modem decides how often it reports,
and Ford may limit how often an app can ask. If Ford answers "too many requests", CarTrak
obeys the wait Ford asks for (at least 30 seconds), says so on the Live page, and then continues.
Lower values than the default can make this happen sooner.

## Data use (per device)
More -> **Data use (this device only)**. The choice is remembered on that device only, so
a phone on a limited plan can use Data saver while your desktop stays Live.

| Mode | What it does |
|---|---|
| **Auto** (default) | Uses Data saver if your phone's own Data Saver is on, otherwise Normal |
| **Normal** | Full detail, refreshes every 30 s while the screen is open |
| **Live** | Updates arrive the moment a reading is saved; paused while the app is in the background |
| **Data saver** | Small summaries, refresh every 5 min, nothing in the background; the trip route and the All data page download only when you tap |

Everything is also compressed (gzip) and uses "unchanged" replies, so a refresh with
nothing new costs almost nothing. What the device keeps for offline use is described below.

## Layouts
- **Phone:** one column, tabs along the bottom.
- **Tablet and desktop:** the home page becomes a grid of two or three columns, trip and charging lists use several columns, and on wide screens the tabs move to a sidebar.

## Settings
| Setting | Default | |
|---|---|---|
| `CARTRAK_POLL_LIVE_SEC` | `10` | Seconds between readings while driving in Road trip mode (minimum 5) |

The normal intervals are in [Configuration](configuration.md).

## Offline
Your server keeps collecting from Ford even when your phone has no signal, so nothing is lost: open the app
again and the history is there. Each device also keeps the last answer of every read-only screen (Home, Trips, Day, Charging,
Statistics, the mileage log and the rest), per car, in its own storage. With no signal those screens open with a small
"Offline. Showing what this device saved" bar instead of an error. Two things are deliberately not kept: screens with
secrets or people (sharing, the sign-in log, Ford link, backups, reports, the Debug report), and anything you change.
Changes say "You are offline" and need a connection. Map pictures you have looked at are kept too (up to 500), so a map seen
before still draws. Everything on the device is wiped when you sign out. Screens you never opened while online have nothing
saved, so open the ones you want in a dead zone beforehand.

## Ford's request limit (why 30 seconds)
Ford only allows an app roughly **one request per endpoint every 30 seconds**; asking sooner returns HTTP 429
("too many requests"). So CarTrak never goes faster than that, even in Road trip mode, serves a recent answer from
memory instead of asking again too soon, adds at most one of the slower endpoints per reading, and after a 429 stops
sending anything and waits (30 s, then 1 min, 2 min... up to 10 min, or longer if Ford says so). For finer detail
than every 30 seconds on a road trip, the phone's own GPS (planned) is the way, not faster polling of Ford.

## Trip time and driving today
The Live screen's **Trip time** tile shows how long the current trip has been going and when it started, and **Driving today** adds up all the time spent driving since midnight (in the car's time zone), including the trip in progress. Both keep counting between readings.

## One screen that follows the car
Live fits the screen without scrolling, on a phone or a tablet on a car mount. The battery ring is always there, and six tiles change with what the car is doing: driving (speed, this trip, trip time, miles to your reserve, range, driving today), charging (power, time to your target, energy added this session, range, energy left, outside temperature) or parked (range, energy left, doors, driving today, outside temperature, odometer). Tap **Time to target** while charging to switch between 80%, 90% and 100%.

## When did the car last report?
"Car reported 6 min ago" at the top of the page, and on Live, is when the car itself last had something new to say, from Ford's update times for its values. It is not when CarTrak last asked Ford: CarTrak asks often, but a parked, sleeping car reports only when it wakes (for example for charging, unlocking or a trip). Press and hold the label (or hover on a computer) to see when CarTrak last checked Ford. If Ford has not answered for a quarter of an hour it adds "Ford not answering", which points to a connection problem rather than a sleeping car; the Status page has the details.

## Remote start
While the car runs from a remote start, Home and Live show how long is left, counted down from the car's own countdown as of its last report. If the car has not reported for a while the time can drift a little. CarTrak only reads this; it cannot start or stop the car.

## Seasonal themes
**More, Display and data, Seasonal themes** dresses the app up for the time of year: Halloween (October), Thanksgiving (November
until Thanksgiving Day), Christmas (from the day after Thanksgiving to December 26), New Year (December 31 and January 1), winter,
Valentine's (February 1 to 14), St. Patrick's (March 1 to 17), Easter (the two weeks before Easter Sunday), spring, the Fourth of
July (July 1 to 4), summer and autumn. A theme changes the accent colour and the battery ring, puts an ornament by the name, and plays
a short effect on Home once a session (bigger on the holiday itself). Wrapped and its infographic use the season's colours.
**Automatic** follows the date; pick one theme to keep it, or **Off**. It is a choice for each device. The effect never plays with the
phone's "reduce motion" setting or in Data saver.

## What's new
After an update, the first time the app opens it shows what changed in every version since this device last opened it. **Got it**
closes it until the next update; **More, About and updates, What's new** keeps the whole list.
