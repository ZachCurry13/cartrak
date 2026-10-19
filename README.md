# CarTrak

A self-hosted dashboard for a Ford electric vehicle (built for the Mustang Mach-E).
It pulls everything Ford's connected-vehicle API provides, stores it all, detects
trips and charging sessions, and lets you label trips **personal** or **work**,
by hand or with rules like "Mon-Fri 8am-5pm is work".

- Mobile-first web app you can **install on your phone** (PWA), light and dark themes
- Every field Ford returns is kept (raw responses plus parsed readings) and browsable
- Trips and charging sessions are detected automatically; routes are drawn without any map service
- Work/personal rules (days, time window, optional "near this place"), judged in the local time where each trip starts; a manual choice always wins
- A business **mileage log** with an odometer check, a **vacation mode** that makes trips personal, and a year-in-review summary
- **Charging** history with cost (including cheaper night rates), your measured charging loss, a **charging curve** per session and fast chargers compared
- **Efficiency insights**: by temperature and speed, and short reasons why a trip used more or less (cold, highway speed, stop and go, climate, braking)
- **Vehicle health**: 12 V and main battery, the true battery level, tire trends and a slow-leak check, parked drain, a data-quality check and a repair tool for a day of trips
- A **road-trip planner**, named places, service reminders, public share links for a trip, and a read-only Home Assistant feed
- **Alerts** on the phone (charging, plug-in reminder, low battery, tires, service due, bumps while parked, left open or unlocked), each device choosing its own
- CSV and spreadsheet export, backup and restore from the app
- A full-screen **Live view** and **Road trip mode** for fast updates while driving, with pushed updates instead of polling
- A per-device **Data saver** mode for limited plans, and layouts for phone, tablet and desktop
- Getting-started guide, the version shown in the app, update notices, and the changelog built in
- Settings in an **Admin page** in the app, or in `config.yaml`, or environment variables
- Built-in login with admin and read-only viewer roles (viewers get a simpler set of tabs and menu)
- One Go binary, SQLite storage, Docker image, TrueNAS and Cloudflare tunnel ready

> CarTrak is an independent open-source project and is not affiliated with or
> endorsed by Ford Motor Company.
>
> **Bring your own Ford developer app.** CarTrak includes no Ford credentials: each person registers
> their own app with Ford and enters its client ID and secret in their own install ([details](docs/ford-setup.md#you-need-your-own-ford-developer-app)).

## Try it without a car
```bash
CARTRAK_PROVIDER=demo go run ./cmd/cartrak
```
Open http://localhost:8080 and create the admin account. The demo provider
simulates parking, driving and charging so you can watch trips appear.

## Install on TrueNAS SCALE (Custom App)

Use **Apps -> Discover Apps -> Custom App** and fill the form. Full field-by-field guide:
**[docs/truenas.md](docs/truenas.md)**. The short version:

| Form section | Set |
|---|---|
| Application name | `cartrak` |
| Image Configuration | Repository `ghcr.io/zachcurry13/cartrak`, Tag `latest`, Pull Policy **always pull** |
| Security Context | Run as user `568`, group `568` (not privileged) |
| Network Configuration | Port: host port e.g. `31000` -> container port `8080`, TCP |
| Portal Configuration | Optional **Web UI** button: HTTP, port `31000` |
| Storage Configuration | Host Path -> `/data` (database, **back it up**) and Host Path -> `/config` (optional `config.yaml`) |
| Environment Variables | None required |

Give the **apps** user (UID 568) read/write access to both folders, install, open the app and:

1. Create the admin account.
2. Go to **More -> Admin settings** and enter your Ford client ID, secret,
   redirect URL and time zone, then **Connect Ford**. ([Ford setup](docs/ford-setup.md): we recommend a
   **separate FordPass account** for CarTrak.)
3. Follow the **Getting started** page (install on your phone, add a work rule, add family).
4. For access away from home (and automatic Ford linking) open **More -> Remote access** and paste a Cloudflare tunnel token; CarTrak runs the connector itself
   ([guide](docs/cloudflare.md)). A separate tunnel app also works.

**Updating:** CarTrak shows an "Update available" banner. In TrueNAS open the app, **Stop**, then **Start**:
it pulls the newest release and keeps your folders. See [Updating](docs/updating.md).

<details><summary>Prefer YAML? (CarTrak and the tunnel in one app)</summary>

Apps -> Custom App -> **Install via YAML**, paste
[deploy/truenas-compose.yaml](deploy/truenas-compose.yaml) and replace each `CHANGE-ME`.
</details>

## Configuration

Settings come from three places. Highest wins:

1. **Environment variables**
2. **`config.yaml`** in the config folder (same names as the variables; CarTrak writes a commented `config.example.yaml` next to it)
3. **Admin settings** in the app (More -> Admin settings), stored in the database

A value set by 1 or 2 shows as *locked* in the Admin page. Everything else (people, trip rules, units,
theme, update checks) is edited in the app and stored in the database.

| Variable | Default | Description | In Admin page |
|---|---|---|---|
| `CARTRAK_PROVIDER` | `fordconnect` | `fordconnect` (FordConnect 2.0, your car), `fordconnect1` (older API) or `demo` (simulated car) | |
| `CARTRAK_ADDR` | `:8080` | Listen address | |
| `CARTRAK_DATA` | `/data` in Docker, `./data` otherwise | Directory for `cartrak.db` | |
| `CARTRAK_CONFIG` | `/config` in Docker, `./config` otherwise | Directory holding `config.yaml` | |
| `CARTRAK_TZ` | `UTC` | Default IANA time zone, used only when the car location is unknown. Trips use the zone where they start, e.g. `America/Chicago` | yes |
| `CARTRAK_GEOCODE_ENABLED` | (on) | `off` stops the mileage log from asking a place-lookup service for city names. See [mileage log](docs/mileage.md) | yes |
| `CARTRAK_PLUG_REMINDER_TIME` / `CARTRAK_PLUG_REMINDER_BELOW` | (9 pm, below 60%) | When the evening Plug in tonight? alert applies; see [notifications](docs/notifications.md) | yes |
| `CARTRAK_GEOCODE_URL` | OpenStreetMap Nominatim | Another Nominatim-compatible lookup service | yes |
| `CARTRAK_MAP_ENABLED` | (on) | `off` turns maps off and draws plain route lines. Tiles come through CarTrak, see [maps](docs/maps.md) | yes |
| `CARTRAK_MAP_TILE_URL` | OpenStreetMap | Another map tile server, with `{z}`, `{x}`, `{y}` in the address | yes |
| `CARTRAK_MAP_ATTRIBUTION` | OpenStreetMap credit | Credit line shown on maps | yes |
| `CARTRAK_POWERTRAIN` | (auto) | Vehicle type: `ev`, `phev`, `hev` or `gas`. Empty works it out from Ford. See [vehicle health](docs/vehicle-health.md) | yes |
| `CARTRAK_TRACK_KEEP_DAYS` | (empty = keep all) | Delete phone GPS points older than this many days. Trips are always kept. See [phone tracking](docs/phone-tracking.md) | yes |
| `FORD_CLIENT_ID` | (empty) | From Ford's developer portal | yes |
| `FORD_CLIENT_SECRET` | (empty) | Keep it private; the Admin page never shows it again | yes |
| `FORD_APPLICATION_ID` | (empty) | Only for the older `fordconnect1` provider; not needed for FordConnect 2.0 | yes |
| `FORD_AUTH_URL` | (built-in) | Ford sign-in (authorize) address, only if Ford documents a different one | yes |
| `FORD_REDIRECT_URL` | address you browse from + `/ford/callback` | Must match the redirect URI registered with Ford | yes |
| `FORD_VIN` | (first authorized) | Choose a vehicle if several are authorized (`fordconnect1` only) | yes |
| `CARTRAK_ADMIN_USER` | `admin` | Optional, for automated installs: username for a pre-created first admin | |
| `CARTRAK_ADMIN_PASS` | (empty) | Optional: if set and no users exist, creates that admin at startup. Leave empty to create the admin in the browser | |
| `CARTRAK_RESET_ADMIN` | (empty) | `1` resets `CARTRAK_ADMIN_USER`'s password to `CARTRAK_ADMIN_PASS` on start (recovery). Remove it afterwards | |
| `CARTRAK_TRUST_PROXY` | `true` | Use `CF-Connecting-IP` / `X-Forwarded-For` / `X-Real-IP` for client IPs (login lockout). Set `false` if the app is reachable without a proxy | |
| `CARTRAK_SESSION_DAYS` | `30` | How long a sign-in lasts | |
| `CARTRAK_POLL_DRIVING_SEC` | `30` | Seconds between Ford readings while driving (minimum 30: Ford limits requests) | |
| `CARTRAK_POLL_CHARGING_SEC` | `120` | ...while charging | |
| `CARTRAK_POLL_LIVE_SEC` | `30` | ...while driving in Road trip mode (minimum 30). See [live view](docs/live-and-data-saver.md) | | |
| `CARTRAK_POLL_PARKED_SEC` | `600` | ...while parked. Lower values may breach Ford's rate limits | |
| `CARTRAK_PRICE_HOME` / `CARTRAK_PRICE_PUBLIC` / `CARTRAK_CURRENCY` | (empty, `$`) | Price per kWh at home and away, and the symbol, for charging costs. See [road trips and charging](docs/road-trips-and-charging.md) | yes |
| `CARTRAK_PRICE_OFFPEAK` / `CARTRAK_OFFPEAK_WINDOW` | (empty) | A cheaper home price and the hours it applies, such as `22:00-06:00` | yes |
| `CARTRAK_CHARGE_EFFICIENCY` | `90` | Share of wall energy that reaches the battery on slow charging; measured from your own sessions once you enter the charger's kWh | yes |
| `CARTRAK_MILEAGE_RATE` | (empty) | Deduction per mile for the mileage log and year in review estimates | yes |
| `CARTRAK_LOW_BATTERY` / `CARTRAK_CHARGE_BUDGET` | (off) | Level for the Low battery alert and a monthly amount for the Charging budget alert | yes |
| `CARTRAK_BUMP_THRESHOLD` | `1.0` | How big a jump on a parked car's motion sensors counts as a bump (or `off`). See [notifications](docs/notifications.md) | yes |

More detail in [docs/configuration.md](docs/configuration.md).

## Security notes
- Data includes your car's location: keep a Cloudflare Access policy, and never publish the app's port directly.
- Passwords are stored hashed (bcrypt); sign-in attempts are rate limited; cookies are `HttpOnly`, `SameSite` and `Secure` over HTTPS.
- The Ford secret and refresh token are never sent to the browser or written to logs. They are stored in the database file in `/data`, so protect and encrypt backups of it.
- Update checks only read GitHub's public release list; turn them off in More -> About.
- Notifications: alerts are encrypted for each device and sent through the push service of its browser maker (Google, Mozilla or Apple), which can see that one was sent but not its content; they never carry a location, and only real push-service addresses are contacted.
- Mileage log: to name cities, the server asks OpenStreetMap Nominatim only when an admin builds a log, with locations rounded to about 100 m, and remembers answers. Switch it off in Settings to send nothing.
- Wind lookup (optional, off by default): when switched on, the server asks Open-Meteo for the wind along a trip, sending only a half-degree map cell (about 50 km) and the date, never an exact position, and remembers the answers.
- Maps: tiles are fetched by the CarTrak server and cached on disk, so browsers never contact the map provider; only signed-in people can load tiles. Switch maps off in Settings for no map requests at all.
- Phone tracking tokens are stored only as a hash and can be removed at any time (More -> Phone tracking); phones can only send data, never read it.

## Documentation
[TrueNAS](docs/truenas.md) · [Cloudflare tunnel](docs/cloudflare.md) · [Ford setup](docs/ford-setup.md) ·
[Configuration](docs/configuration.md) · [Road trips and charging](docs/road-trips-and-charging.md) · [Notifications](docs/notifications.md) · [Mileage log](docs/mileage.md) · [Maps](docs/maps.md) · [Car data](docs/car-data.md) · [Vehicle health](docs/vehicle-health.md) · [Backup and security](docs/backup.md) · [Places and maintenance](docs/places-and-maintenance.md) · [Sharing](docs/sharing.md) · [Home Assistant](docs/home-assistant.md) · [Phone tracking](docs/phone-tracking.md) · [Live view and data saver](docs/live-and-data-saver.md) · [Updating](docs/updating.md) · [Troubleshooting](docs/troubleshooting.md)

## Development
```bash
go test ./... && go vet ./...
```
Web files in `internal/web/static` are embedded in the binary; restart after editing. They are served from
addresses that carry the version (`/v0.26.0/js/app.js`) and the service worker's cache is named after the
version, so nothing needs bumping by hand and no cache can keep an old copy.

Branches: work happens on one short-lived `feature/vX.Y` branch, which is fast-forwarded into `main` at
release and then deleted. `main`, the version tags and the GitHub Releases are the whole history.

## License
MIT. See [LICENSE](LICENSE).
