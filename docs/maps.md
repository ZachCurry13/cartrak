# Maps

CarTrak shows real maps in three places:
- **Trip page:** the route on a map with start (green) and end (red) markers. It uses your phone's GPS track when there is one, otherwise Ford's readings.
- **Home:** a small map of where the car is.
- **Trips, Map:** every trip in the chosen range on one map, work in amber and personal in blue. Tap a route to open the trip.

Drag to move, use the + and − buttons, the mouse wheel or a two-finger pinch to zoom.

## Privacy: who sees what
Your phone or computer **never contacts the map provider**. Map tiles are pictures of small squares of the map. CarTrak's own server downloads each tile from OpenStreetMap the first time it is needed, keeps a copy on disk (`tiles` inside the data folder), and passes it on. So:
- OpenStreetMap sees only your server's address and which squares were asked for, never your phone, your account or which trip you opened.
- Repeat views come from the copy on disk and are fast, and still work for a while if the map provider is down.
- Only signed-in people can load tiles, so the server is not an open proxy. Only real map images are passed on.
- CarTrak's strict content policy is unchanged, because the tiles come from CarTrak itself.

If you would rather not use maps at all, set **More, Settings, Maps** to Off. CarTrak then draws the plain route lines it used before and makes no map requests.

## Settings (More, Settings)
| Setting | Environment variable | Meaning |
|---|---|---|
| Maps | `CARTRAK_MAP_ENABLED` | Empty is on; `off` turns maps off |
| Map tile server (advanced) | `CARTRAK_MAP_TILE_URL` | Another provider's address, with `{z}`, `{x}` and `{y}` in it. Must be https (or http on localhost). Default: OpenStreetMap |
| Road estimate for trips without phone GPS | `CARTRAK_ROUTE_LOOKUP` | Empty is off; `on` draws an estimated road for trips the car recorded with only a few positions, dashed, and never over a phone GPS track |
| Routing service address (advanced) | `CARTRAK_ROUTE_URL` | An OSRM-compatible server. Empty uses the public demo server (light use only). Your own keeps the positions off the internet |
| Map credit (advanced) | `CARTRAK_MAP_ATTRIBUTION` | The credit line shown on maps. Change it together with the tile server |

## Estimated roads
A car often reports only where a trip started and ended, so its route is a straight line across lakes and fields. With the road estimate switched on (More, Settings), CarTrak asks a routing service for the road between the positions the car did record (up to eight, so the estimate stays on the road you used), checks that its length is believable (against the odometer, or against the straight line from start to end when the odometer is plainly short), and draws it dashed on the trip. It is a guess at the likely road, not where the car went: only a phone GPS track shows that, and a phone track always wins. The answer is kept, so the service is asked once per trip.

The Trips list also asks for a few missing roads in the background each time it is opened, and making a share link finds the roads of its trips first. On the Trips list, a dashed thumbnail is a line between the few places the car reported, until its road is found. Older trips: open More, Trips and press **Estimate their roads** to do them a few at a time (the routing service is queried slowly on purpose). Switching the estimate off does not delete estimates already made.

## Trip replay
On a trip page, **Replay this trip** plays the trip back on its map: a dot drives the route while the clock, speed, battery and the distance so far follow it. Drag the slider to jump to any moment, and choose 10×, 30×, 60× or 120×. A phone GPS track gives a smooth replay; with only the car's own positions, which it reports now and then, the dot moves in straight lines between them. The distance follows the route and ends at the trip's recorded distance.

## Day view
Open **Day view** from the Trips page (or **Map of this day** in the History feed) to see a whole day at once: each trip is its own color, stops between trips are numbered markers that match the list below the map, and a written summary gives the day's driving, stops, charging, energy and cost. Move between days with the arrows or the date box. A phone GPS track is used where there is one, then a kept estimated road, then the car's own positions (which may be only a few). Tap a drive or a charging session in the list to open its details right there instead of leaving the day.

Each row of the day, and of the History feed, says where it happened: a drive as "from → to", a stop or a charge by its place. A place you named wins; otherwise it is the city, looked up in the background the first time (a drive that began and ended in the same town reads "Around" it). Drives also show their average speed and efficiency, and charges their average power and what a kWh cost (from your receipt, or an estimate from your price). Each row shows the battery at its start and end ("98 → 51%"), since on a road trip what you arrive with matters most. On a day with charging on the road, drives between two charges that make up more than one drive are summed up as a leg, a charge says how long the whole stop was when you stayed longer than it charged, and the car finishing its charge at home just before you leave folds into the first drive as "Left home at 98%" (and is not counted as a charge in the summary). On such a day the summary also says how much of the time on the road went on charging and the longest stretch between charges. Open a public charge whose cost is an estimate and you can give that charger its own price per kWh ("What does a kWh cost here?"): it is saved as a place, and every charge there is worked out with it.

## Good to know
- OpenStreetMap's tile server is meant for light use and asks apps to identify themselves; CarTrak sends `CarTrak/<version>` with a link to its project page and keeps copies so each tile is downloaded rarely. For heavy or commercial use choose a provider that allows it and set it above.
- The credit "© OpenStreetMap contributors" appears on every map and links to the copyright page.
- Cached tiles are reused for 30 days and deleted after 60 days without being needed. The map folder is safe to delete at any time; it refills as you look at maps.
- The server needs to be able to reach the internet for new tiles. With no internet, only tiles already cached appear and the rest of the map area stays blank (routes and markers still draw).
- On a metered connection Data saver skips the Home map; the trip page already waits for you to load the route.
