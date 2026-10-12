# Handoff

## State
Early scaffold on `main`; no release yet. Public repo `ZachCurry13/cartrak` (to be created). Rules are in CLAUDE.md.

## Decisions
- Standalone Go app, SQLite (modernc, no CGO), embedded PWA, admin + viewer roles.
- Two switchable Ford providers (`fordconnect` official, `fordpass` app API); both unverified against a real car.
- Trips are auto-classified personal/work by rules (days + time window, optional geofence); manual override always wins.
- Deploy: Docker image on GHCR + TrueNAS custom-app compose YAML with a `cloudflared` sidecar.

## Next immediate steps
1. FordPass provider (auth flow per marq24/ha-fordpass), then FordConnect.
2. Poller, trip and charge detection, rules engine.
3. HTTP API + login, PWA UI, Dockerfile, docker.yml, TrueNAS YAML.

## Known gaps
- Ford endpoints unverified; user's credentials live in a git-excluded `.secrets/` folder.
