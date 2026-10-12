# CarTrak

A self-hosted dashboard for a Ford electric vehicle (built for the Mustang Mach-E).
It pulls everything Ford's connected-vehicle API provides, stores it all, detects
trips and charging sessions, and lets you label trips **personal** or **work**,
by hand or with rules like "Mon-Fri 8am-5pm is work".

- Mobile-first web app you can **install on your phone** (PWA), light and dark themes
- Every field Ford returns is kept (raw responses plus parsed readings), and browsable
- Trips and charging sessions are detected automatically; routes drawn without any map service
- Work/personal rules (days, time window, optional "near this place"); manual choices always win
- CSV export, work/personal mileage totals
- Built-in login with admin and read-only viewer roles
- Getting-started guide, version shown in the app, update notices, and the changelog built in
- One Go binary, SQLite storage, Docker image, TrueNAS and Cloudflare tunnel ready

> CarTrak is an independent open-source project and is not affiliated with or
> endorsed by Ford Motor Company.

## Try it without a car
```bash
CARTRAK_PROVIDER=demo go run ./cmd/cartrak
```
Open http://localhost:8080 and create the admin account. The demo provider
simulates parking, driving and charging so you can see trips appear.

## Connect your Ford vehicle
CarTrak uses the FordConnect API, which needs an app registered in Ford's
developer portal (client ID, client secret, Application-Id and a redirect URI).
Then sign in to CarTrak as admin and choose More -> Connect Ford. Ford's own
page handles your login; CarTrak only stores a refresh token.

## Configuration (environment variables or `config.yaml`)
| Variable | Meaning |
|---|---|
| `CARTRAK_PROVIDER` | `fordconnect` or `demo` |
| `CARTRAK_DATA` | data folder (default `./data`, `/data` in Docker) |
| `CARTRAK_CONFIG` | config folder holding `config.yaml` (default `./config`, `/config` in Docker) |
| `CARTRAK_ADDR` | listen address (default `:8080`) |
| `CARTRAK_TZ` | IANA time zone used by work-hour rules (default `UTC`) |
| `FORD_CLIENT_ID`, `FORD_CLIENT_SECRET`, `FORD_APPLICATION_ID` | from the Ford developer portal |
| `FORD_REDIRECT_URL` | must match the redirect URI registered with Ford |
| `FORD_VIN` | optional vehicle id; default is the first authorized vehicle |
| `CARTRAK_ADMIN_PASS` | optional; otherwise create the admin on first visit |
| `CARTRAK_POLL_DRIVING_SEC` / `_CHARGING_SEC` / `_PARKED_SEC` | polling intervals (30 / 120 / 600) |

## Deploying
Guides: [TrueNAS install](docs/truenas.md) · [Cloudflare tunnel](docs/cloudflare.md) · [Ford setup](docs/ford-setup.md) · [Configuration](docs/configuration.md) · [Updating](docs/updating.md) · [Troubleshooting](docs/troubleshooting.md)

TrueNAS YAML: [deploy/truenas-compose.yaml](deploy/truenas-compose.yaml). Two folders are mapped: `data` (database, back it up) and `config` (`config.yaml`).

## Development
```bash
go test ./... && go vet ./...
```
Web files in `internal/web/static` are embedded in the binary; restart after editing.
When changing them, bump the cache name in `sw.js`.

## License
MIT. See [LICENSE](LICENSE).
