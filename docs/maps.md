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
| Map credit (advanced) | `CARTRAK_MAP_ATTRIBUTION` | The credit line shown on maps. Change it together with the tile server |

## Good to know
- OpenStreetMap's tile server is meant for light use and asks apps to identify themselves; CarTrak sends `CarTrak/<version>` with a link to its project page and keeps copies so each tile is downloaded rarely. For heavy or commercial use choose a provider that allows it and set it above.
- The credit "© OpenStreetMap contributors" appears on every map and links to the copyright page.
- Cached tiles are reused for 30 days and deleted after 60 days without being needed. The map folder is safe to delete at any time; it refills as you look at maps.
- The server needs to be able to reach the internet for new tiles. With no internet, only tiles already cached appear and the rest of the map area stays blank (routes and markers still draw).
- On a metered connection Data saver skips the Home map; the trip page already waits for you to load the route.
