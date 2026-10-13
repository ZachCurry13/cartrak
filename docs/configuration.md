# Configuration

Settings come from three places. Highest wins:

1. **Environment variables** (for example in the TrueNAS YAML)
2. **`config.yaml`** in the config folder (`/config` in Docker, `./config` otherwise)
3. Built-in defaults

`config.yaml` uses the same names as the environment variables. CarTrak writes
a fully commented [`config.example.yaml`](../internal/config/config.example.yaml)
into the config folder every time it starts; copy it to `config.yaml`.
Restart the app after editing.

| Key | Meaning | Default |
|---|---|---|
| `CARTRAK_PROVIDER` | `fordconnect` (FordConnect 2.0, your car), `fordconnect1` (older API) or `demo` (simulated) | `fordconnect` |
| `CARTRAK_TZ` | IANA time zone for work-hour rules, e.g. `America/Chicago` | `UTC` |
| `CARTRAK_POWERTRAIN` | vehicle type: `ev`, `phev`, `hev` or `gas`; empty works it out from Ford (garage record, then the data the car reports) | auto |
| `CARTRAK_TRACK_KEEP_DAYS` | delete phone GPS points older than this many days (empty keeps everything; trips are always kept) | keep all |
| `FORD_CLIENT_ID` / `FORD_CLIENT_SECRET` | from Ford's developer portal | none |
| `FORD_APPLICATION_ID` | sent as `Application-Id`; only for the older `fordconnect1` provider | none |
| `FORD_AUTH_URL` | Ford sign-in (authorize) address, only if Ford documents a different one | built-in |
| `FORD_REDIRECT_URL` | must match the redirect URI registered with Ford | none |
| `FORD_VIN` | choose a vehicle if several are authorized (`fordconnect1` only) | first authorized |
| `CARTRAK_ADMIN_USER` / `CARTRAK_ADMIN_PASS` | create the admin at startup (otherwise use first-run setup) | none |
| `CARTRAK_RESET_ADMIN` | `1` resets the password of `CARTRAK_ADMIN_USER` to `CARTRAK_ADMIN_PASS` on start (recovery) | off |
| `CARTRAK_TRUST_PROXY` | believe proxy headers (`CF-Connecting-IP`, `X-Forwarded-For`, `X-Real-IP`) for client IPs | `true` |
| `CARTRAK_SESSION_DAYS` | how long a sign-in lasts | `30` |
| `CARTRAK_POLL_DRIVING_SEC` | seconds between readings while driving (minimum 30: Ford's limit) | `30` |
| `CARTRAK_POLL_CHARGING_SEC` | while charging | `120` |
| `CARTRAK_POLL_LIVE_SEC` | while driving in Road trip mode (minimum 30) | `30` |
| `CARTRAK_POLL_PARKED_SEC` | while parked (minimum 60) | `600` |
| `CARTRAK_ADDR` | listen address | `:8080` |
| `CARTRAK_DATA` | data folder | `./data` (`/data` in Docker) |
| `CARTRAK_CONFIG` | config folder | `./config` (`/config` in Docker) |

Things set inside the app (people, rules, units, theme, update-check) are
stored in the database in `/data`, not in `config.yaml`.

## The Admin settings page
More -> Admin settings (admins only) edits these keys and stores them in the
database: `FORD_CLIENT_ID`, `FORD_CLIENT_SECRET`, `FORD_APPLICATION_ID`,
`FORD_REDIRECT_URL`, `FORD_VIN`, `CARTRAK_TZ`, `CARTRAK_POWERTRAIN` and `CARTRAK_TRACK_KEEP_DAYS`. Changes apply right away, no restart.

- Precedence is environment variable, then `config.yaml`, then the Admin page. A key set by the first two shows as **locked** on the page: change it where it is set.
- The Ford secret is write-only: the page never shows it again, and a blank secret field keeps the saved one.
- If the redirect URL is empty, CarTrak uses the address you are browsing from plus `/ford/callback` when you press Connect Ford.
- Saved values live in `cartrak.db` in the `data` folder, so protect backups of it.

## Time zones
CarTrak works out each trip's time zone from **where the trip starts** (boundary data is built in,
so this needs no internet). Work and personal rules, and the times shown for the trip, use that local time:
"Mon-Fri 8am-5pm" means 8 to 5 wherever you are driving. The `CARTRAK_TZ` setting (Admin settings, "Default
time zone") is only the fallback when a reading has no location. Each trip stores its zone, so old trips keep
the zone they were recorded in. On a device, More -> **Show trip times in** chooses between the trip's local
time (default) and the device's own time.
