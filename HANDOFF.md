# Handoff

## State
v0.1.0 is released (tag v0.1.0, image `ghcr.io/zachcurry13/cartrak:0.1.0` and `:latest`, public). `feature/v0.2` holds v0.2.0 (finished, pushed, NOT released).
Public repo: https://github.com/ZachCurry13/cartrak (MIT). Rules are in CLAUDE.md.

## Decisions
- Standalone Go app, SQLite (modernc, no CGO), embedded vanilla-JS PWA, admin + viewer roles.
- Ford: FordConnect v1 (owner link login, refresh token stored in DB). FordPass-style fallback deferred.
- Trips auto-classified personal/work by rules (days + time window, optional geofence); manual always wins.
- Ford-inspired palette only (no logos, fonts, or Ford name in branding); disclaimer in More page.
- Deploy: GHCR image via docker.yml, TrueNAS compose YAML with a cloudflared sidecar.
- comma 4 + StarPilot: planned as a separate data source; the device pushes drive summaries/GPS to CarTrak over the tunnel with a token (most private; Ford never involved). Ideas: openpilot engagement % per trip, disengagements, efficiency openpilot vs manual, device health.

## Next immediate steps
1. User updates TrueNAS to 0.3.1 (Stop/Start) and retries Connect Ford; fix whatever Ford returns (link/token/field names are still unverified live).
2. Planned (awaiting user priorities): live/roadtrip mode (SSE push, adaptive polling), responsive PC/tablet layout, per-device data-saver mode, ingest API with device tokens (shared by comma push and phone OBD-II recorder), more FordConnect endpoints.
2. Verify against real Ford: token URL, link URL, api-version, field names (see internal/ford/fordconnect.go, parse.go).
2. Build the comma push endpoint + device script; add FordPass fallback if wanted.

## Known gaps
- Ford endpoints unverified live. Docker image untested locally (no Docker); CI builds it.
- Local-only files (git-excluded): `.secrets/` (Ford creds + data dictionary PDF), `.claude/`.
- Demo launch config: `.claude/launch.json` (git-excluded).

## v0.2 notes
- TrueNAS update = custom YAML with `pull_policy: always`; user does Stop then Start. A real catalog submission was offered but not chosen.
- Update check: daily GitHub releases call, on by default, toggle in More -> About (kv `update_check`). `CARTRAK_UPDATE_URL` env is a testing override.
- Config precedence: env > /config/config.yaml > defaults. `config.example.yaml` is rewritten on every start.
- Password recovery: CARTRAK_RESET_ADMIN=1 with CARTRAK_ADMIN_USER/PASS.
- Changelog is embedded from CHANGELOG.md (root embed.go); the release workflow uses the `## X.Y.Z` section as notes, so every release needs that heading.

## FordConnect 2.0 (v0.5)
- The user's Ford app is FordConnect **2.0** (Postman collection `FordConnect-2.0.postman_collection.json`, kept in git-excluded `.secrets/`). Provider `fordconnect` = 2.0, `fordconnect1` = the older v1 API (kept), `demo`.
- 2.0 facts from the collection: token + refresh at `https://api.vehicle.ford.com/dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token?p=B2C_1A_FCON_AUTHORIZE` (form: grant_type, code/refresh_token, redirect_uri, client_id, client_secret, scope=`<client_id> offline_access openid`); data via bearer GET `https://api.vehicle.ford.com/fcon-query/v1/` + `telemetry | garage | vehicle-health/alerts | wallbox | electric/departure-times | electric/charge-schedules | fccs` (fccs needs a `chargingStationId` header + startDate/endDate; not polled yet).
- UNVERIFIED: the authorize (sign-in) URL (guessed from the token URL pattern; override with FORD_AUTH_URL / Admin settings), and every response shape (the collection has no examples). `ParseState` handles `{name,value}` metric lists via metric aliases, but the real shape must be confirmed.
- Tool for this: Admin settings -> Ford diagnostics (POST /api/admin/ford-probe) returns each endpoint's status and body with location/VIN values redacted. Ask the user to run it and paste the output, then build dashboard cards (tires, battery temp, etc.) from the real shape.
- Slow endpoints (garage, alerts, wallbox, schedules) are read every 30 min; failures never block telemetry (`_endpointErrors` in raw).

## Other planned work
- Phone OBD-II recorder (Android Chrome + BLE adapter) and comma 4 push, via a shared ingest API with device tokens.

## FordConnect 2.0 sign-in: how it really works (found in v0.8.1 from marq24/ha-fordconnect-query)
- Sign-in START: GET https://api.vehicle.ford.com/fcon-public/v1/auth/init?response_type=code&client_id=..&redirect_uri=..&state=<=16 chars> -> 302 to login.ford.com (Ford login + vehicle consent) -> redirect to the registered redirect_uri with code+state. NO scope on the start. state MUST be <=16 chars.
- Token: POST .../dah2vb2cprod.onmicrosoft.com/oauth2/v2.0/token?p=B2C_1A_FCON_AUTHORIZE (code, grant_type, redirect_uri, client_id, client_secret; scope optional, CarTrak tries with then without on 400).
- The B2C_1A_FCON_AUTHORIZE *authorize* endpoint is NOT a sign-in start (always AADB2C90075); CarTrak uses it only to check which redirect URIs are registered (accepted = proceeds to AADB2C90075, rejected = AADB2C90006 mismatch).
- Newly registered apps can take ~2h to deploy; redirect-address edits also take time (the user's with-path entry became accepted hours after being added).
- Telemetry shape: vehicles[].metrics.<camelCase>.value, e.g. xevBatteryRange, position.value.location.{lat,lon,alt}, heading, compassDirection, doorLockStatus, hoodStatus, tire/indicator lists, customMetrics.*. Read more in marq24's fordpass_handler.py. Parser matches names ignoring case and underscores.

## Ford rate limit (v0.8.2)
- Real behaviour seen: right after linking, all 6 endpoints returned 429 (~230-250 byte bodies). Users of this API report the 2nd call to the same endpoint within seconds gets 429 and ~30 s spacing avoids it (evcc discussion #25639; HA integration defaults to a 60 s scan, minimum 30 s).
- CarTrak: internal/ford/connect2_throttle.go (30 s per-endpoint gap with cached answers, global holdUntil after a 429, one slow endpoint per reading), poller backoff 30 s doubling to 10 min (rateLimitWait), config floors (secsMin). The exact 429 body/Retry-After from Ford is still unseen: ask the user for Copy responses after waiting a few minutes.
