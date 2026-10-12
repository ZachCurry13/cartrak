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
| `CARTRAK_PROVIDER` | `fordconnect` (real car) or `demo` (simulated) | `fordconnect` |
| `CARTRAK_TZ` | IANA time zone for work-hour rules, e.g. `America/Chicago` | `UTC` |
| `FORD_CLIENT_ID` / `FORD_CLIENT_SECRET` | from Ford's developer portal | none |
| `FORD_APPLICATION_ID` | sent as `Application-Id` | none |
| `FORD_REDIRECT_URL` | must match the redirect URI registered with Ford | none |
| `FORD_VIN` | choose a vehicle if several are authorized | first authorized |
| `CARTRAK_ADMIN_USER` / `CARTRAK_ADMIN_PASS` | create the admin at startup (otherwise use first-run setup) | none |
| `CARTRAK_RESET_ADMIN` | `1` resets the password of `CARTRAK_ADMIN_USER` to `CARTRAK_ADMIN_PASS` on start (recovery) | off |
| `CARTRAK_POLL_DRIVING_SEC` | seconds between readings while driving | `30` |
| `CARTRAK_POLL_CHARGING_SEC` | while charging | `120` |
| `CARTRAK_POLL_PARKED_SEC` | while parked | `600` |
| `CARTRAK_ADDR` | listen address | `:8080` |
| `CARTRAK_DATA` | data folder | `./data` (`/data` in Docker) |
| `CARTRAK_CONFIG` | config folder | `./config` (`/config` in Docker) |

Things set inside the app (people, rules, units, theme, update-check) are
stored in the database in `/data`, not in `config.yaml`.
