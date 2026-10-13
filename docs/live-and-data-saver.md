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
nothing new costs almost nothing. Nothing is cached offline except the app files.

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
again and the history is there. The app also remembers the last readings on each device, so Home opens with a
clear "You are offline" banner instead of an error. Anything not yet saved on the device (other pages) needs a connection.

## Ford's request limit (why 30 seconds)
Ford only allows an app roughly **one request per endpoint every 30 seconds**; asking sooner returns HTTP 429
("too many requests"). So CarTrak never goes faster than that, even in Road trip mode, serves a recent answer from
memory instead of asking again too soon, adds at most one of the slower endpoints per reading, and after a 429 stops
sending anything and waits (30 s, then 1 min, 2 min... up to 10 min, or longer if Ford says so). For finer detail
than every 30 seconds on a road trip, the phone's own GPS (planned) is the way, not faster polling of Ford.
