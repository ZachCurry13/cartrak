# CarTrak

A self-hosted dashboard for a Ford electric vehicle (built for the Mustang Mach-E).
It pulls everything Ford's connected-vehicle API provides, stores it all, detects
trips and charging sessions, and lets you label trips **personal** or **work**,
by hand or with rules like "Mon-Fri 8am-5pm is work".

- Mobile-first web app you can **install on your phone** (PWA), light and dark themes
- Every field Ford returns is kept (raw responses plus parsed readings) and browsable
- Trips and charging sessions are detected automatically; routes are drawn without any map service
- Work/personal rules (days, time window, optional "near this place"); a manual choice always wins
- CSV export and work/personal mileage totals
- Getting-started guide, the version shown in the app, update notices, and the changelog built in
- Built-in login with admin and read-only viewer roles
- One Go binary, SQLite storage, Docker image, TrueNAS and Cloudflare tunnel ready

> CarTrak is an independent open-source project and is not affiliated with or
> endorsed by Ford Motor Company.

## Try it without a car
```bash
CARTRAK_PROVIDER=demo go run ./cmd/cartrak
```
Open http://localhost:8080 and create the admin account. The demo provider
simulates parking, driving and charging so you can watch trips appear.

## Install on TrueNAS SCALE (Custom App)

You need TrueNAS SCALE with Apps enabled, and for remote access a Cloudflare
account with a domain. Menu names can differ slightly between TrueNAS versions.
Longer guides: [TrueNAS](docs/truenas.md) · [Cloudflare tunnel](docs/cloudflare.md) · [Ford setup](docs/ford-setup.md).

**1. Create two folders** in a dataset (for example `apps/cartrak/data` and `apps/cartrak/config`)
and give the **apps** user (UID 568) read/write access:

| Folder in the container | What it holds | Back it up? |
|---|---|---|
| `/data` | the database: every reading, trip, charge, rule and user | **Yes** |
| `/config` | `config.yaml` with your settings and Ford credentials | Yes, keep it private |

**2. Install:** Apps -> Discover Apps -> **Custom App** -> **Install via YAML**, name it `cartrak`, paste the
YAML below, and replace each `CHANGE-ME` (your two folder paths, your time zone, the Cloudflare tunnel token).

```yaml
services:
  cartrak:
    image: ghcr.io/zachcurry13/cartrak:latest
    pull_policy: always              # Stop then Start in TrueNAS pulls the newest release
    restart: unless-stopped
    environment:
      CARTRAK_PROVIDER: fordconnect          # "demo" tries the app with a simulated car
      CARTRAK_TZ: UTC                        # CHANGE-ME: your IANA time zone, e.g. America/Chicago
    volumes:
      - /mnt/POOL/apps/cartrak/data:/data        # CHANGE-ME
      - /mnt/POOL/apps/cartrak/config:/config    # CHANGE-ME
  cloudflared:
    image: cloudflare/cloudflared:latest
    pull_policy: always
    restart: unless-stopped
    command: tunnel --no-autoupdate run
    environment:
      TUNNEL_TOKEN: CHANGE-ME                # Cloudflare Zero Trust -> Networks -> Tunnels
    depends_on:
      - cartrak
```
The same file with comments is [deploy/truenas-compose.yaml](deploy/truenas-compose.yaml).

**3. Cloudflare:** create a tunnel, add a public hostname pointing at `http://cartrak:8080`
([details](docs/cloudflare.md)), and add a Cloudflare Access policy for a second lock.

**4. Settings:** on first start CarTrak writes `config.example.yaml` into the `config` folder.
Copy it to `config.yaml`, fill in your Ford details ([Ford setup](docs/ford-setup.md): we recommend a
**separate FordPass account** for CarTrak) and restart the app.

**5. First run:** open your hostname, create the admin account, and follow the **Getting started** page
(connect Ford, install on your phone, add a work rule, add family).

**Updating:** CarTrak shows an "Update available" banner. In TrueNAS open the app, **Stop**, then **Start**:
it downloads the newest release and keeps your folders. See [Updating](docs/updating.md).

## Configuration

Process-level settings are environment variables or keys in `config.yaml` (same names; the
environment wins). Everything else (people, trip rules, units, theme, update checks) is edited in the app
and stored in the database.

| Variable | Default | Description |
|---|---|---|
| `CARTRAK_PROVIDER` | `fordconnect` | `fordconnect` (your car) or `demo` (simulated car) |
| `CARTRAK_ADDR` | `:8080` | Listen address |
| `CARTRAK_DATA` | `/data` in Docker, `./data` otherwise | Directory for `cartrak.db` |
| `CARTRAK_CONFIG` | `/config` in Docker, `./config` otherwise | Directory holding `config.yaml` |
| `CARTRAK_TZ` | `UTC` | IANA time zone used by work-hour rules, e.g. `America/Chicago` |
| `FORD_CLIENT_ID`, `FORD_CLIENT_SECRET` | (empty) | From Ford's developer portal. Keep the secret private |
| `FORD_APPLICATION_ID` | (empty) | Sent to Ford as `Application-Id` |
| `FORD_REDIRECT_URL` | (empty) | Must match the redirect URI registered with Ford, e.g. `https://cartrak.example.com/ford/callback` |
| `FORD_VIN` | (first authorized) | Choose a vehicle if several are authorized |
| `CARTRAK_ADMIN_USER` | `admin` | Optional, for automated installs: username for a pre-created first admin |
| `CARTRAK_ADMIN_PASS` | (empty) | Optional: if set and no users exist, creates that admin at startup. Leave empty to create the admin in the browser |
| `CARTRAK_RESET_ADMIN` | (empty) | `1` resets `CARTRAK_ADMIN_USER`'s password to `CARTRAK_ADMIN_PASS` on start (recovery). Remove it afterwards |
| `CARTRAK_TRUST_PROXY` | `true` | Use `CF-Connecting-IP` / `X-Forwarded-For` / `X-Real-IP` for client IPs (login lockout). Set `false` if the app is reachable without a proxy |
| `CARTRAK_SESSION_DAYS` | `30` | How long a sign-in lasts |
| `CARTRAK_POLL_DRIVING_SEC` | `30` | Seconds between Ford readings while driving |
| `CARTRAK_POLL_CHARGING_SEC` | `120` | ...while charging |
| `CARTRAK_POLL_PARKED_SEC` | `600` | ...while parked. Lower values may breach Ford's rate limits |

More detail in [docs/configuration.md](docs/configuration.md).

## Security notes
- Data includes your car's location: keep the Cloudflare Access policy, and never publish port 8080 directly.
- Passwords are stored hashed (bcrypt); sign-in attempts are rate limited; cookies are `HttpOnly`, `SameSite` and `Secure` over HTTPS.
- The Ford secret and refresh token are never sent to the browser or written to logs.
- Update checks only read GitHub's public release list; turn them off in More -> About.

## Documentation
[TrueNAS](docs/truenas.md) · [Cloudflare tunnel](docs/cloudflare.md) · [Ford setup](docs/ford-setup.md) ·
[Configuration](docs/configuration.md) · [Updating](docs/updating.md) · [Troubleshooting](docs/troubleshooting.md)

## Development
```bash
go test ./... && go vet ./...
```
Web files in `internal/web/static` are embedded in the binary; restart after editing, and bump the cache
name in `sw.js` when they change.

## License
MIT. See [LICENSE](LICENSE).
