# Handoff

## State
Released through v0.10.0 (tag v0.10.0, image ghcr.io/zachcurry13/cartrak:0.10.0 and :latest, public; 2026-10-13). v0.9.0 includes everything built in the 0.9 cycle (status screen, dashboard cards, phone tracking, vehicle health, maps, mileage log) and the held v0.8.2 fixes. feature/v0.11 is the next branch. Releases need the user's explicit OK each time.
Public repo: https://github.com/ZachCurry13/cartrak (MIT). Rules are in CLAUDE.md.

## Decisions
- Standalone Go app, SQLite (modernc, no CGO), embedded vanilla-JS PWA, admin + viewer roles.
- Ford: FordConnect v1 (owner link login, refresh token stored in DB). FordPass-style fallback deferred.
- Trips auto-classified personal/work by rules (days + time window, optional geofence); manual always wins.
- Ford-inspired palette only (no logos, fonts, or Ford name in branding); disclaimer in More page.
- Deploy: GHCR image via docker.yml, TrueNAS compose YAML with a cloudflared sidecar.
- comma 4 + StarPilot: planned as a separate data source; the device pushes drive summaries/GPS to CarTrak over the tunnel with a token (most private; Ford never involved). Ideas: openpilot engagement % per trip, disengagements, efficiency openpilot vs manual, device health.

## Next immediate steps
1. Web Push alerts are RELEASED in 0.10.0 (see docs/notifications.md; not yet tried on a real phone): push.Service sends via webpush-go with VAPID keys in kv; alerts.Engine compares successive readings (hook p.OnReading) and Tick checks silence; subscriptions in push_subs; only googleapis/mozilla/apple/windows push hosts are contacted. Next: the user tries it on a real phone over the https address, then release 0.10.0 if OK.
2. Maps for trips are BUILT (decisions made with the user: server-side OSM tile proxy with disk cache in <data>/tiles, maps on trip page + Home location card + Trips Map view, own slippy map js/map.js, no third-party library, settings CARTRAK_MAP_ENABLED/TILE_URL/ATTRIBUTION). Not yet checked against real OpenStreetMap tiles (tested with a local fake tile server); first real use will show whether OSM answers the CarTrak User-Agent. Docs: docs/maps.md.
2b. Mileage log is BUILT (docs/mileage.md): internal/geocode (Nominatim client, 1 req/s, rounded to 3 decimals, places cache table), internal/mileage (rows, ISO weeks), /api/mileage (admin only), pages/mileage.js. The user's sheet format: Date (YYYY-MM-DD), Origin City, Destination City (comma list, each city once, origin left out), Description (Customer Visits), Total Miles Driven, then Total Mileage Recorded. The user asked for weeks 26-28 but CarTrak only has data since ~Oct 12 2026 (ISO weeks 26-28 are Jun 22-Jul 12), so earlier weeks are empty.
3. Then Web Push alerts (charge complete, plug in/out, low tire, unlocked, trip start/end, low 12 V).
4. Optional: read-only departure times and charge schedules card (the data is already pulled; the Mach-E lists 14 departure slots, all off).
5. Ask the user (AskUserQuestion) before any release; decide whether v0.8.2 ships alone or inside 0.9.0.
6. Later: comma 4 push endpoint (reuse the device-token ingest), phone OBD-II recorder (real DTCs, cell voltages), FordPass control module (off by default, admin-only, confirmations, audit log) only if the user wants commands.
7. Open questions from the user: (a) dad using the same server with his own account and car needs multi-vehicle/multi-owner support (not built; the single Ford link and shared data make every account see one car). Plan agreed: first test whether Ford lets a second FordPass account authorize through the user's developer app, using a throwaway copy (cartrak.exe on port 18081, data in data/fordtest, git-excluded; the user enters their own client ID/secret there, dad signs in at Ford, then paste the http://localhost return address on the Connect page step 2). Then decide between a second CarTrak copy for dad or building multi-user. (b) Android Auto: not possible for web apps; needs a native Android app using the Android for Cars App Library (categories such as POI, IoT), sideloadable via Android Auto developer settings; deferred.

## What the real Ford data looks like (from a real Mach-E; no VIN, location or mileage kept here)
- Layout: `telemetry.metrics.<name>.value` (no vehicles array), each metric also has `updateTime` and `oemCorrelationId`; list metrics are `<name>[i].value` with vehicleWheel / vehicleDoor / vehicleSide. `garage.engineType` is BEV for the Mach-E (authoritative vehicle type). `alerts.VehicleAlertList` holds a placeholder "No Data" (StatusCode 4040) when there are no alerts.
- A battery car also reports oilLifeRemaining=100, hybridVehicleModeStatus, engineCoolantTemp, engineSpeed and tripFuelEconomy=0: never treat these as proof of an engine. `ambientTemp` reads 0 while `outsideTemperature` is right.
- Values update at different times: the xev* battery metrics were 3 days older than the rest (the car reports them when it wakes), so show ages. `batteryStateOfCharge` (PRIMARY_BATTERY) is the 12 V battery's charge; `xevBatteryStateOfCharge` is the main battery as shown on the dash and `xevBatteryActualStateOfCharge` the cells (86.5 vs 91.5). xevBatteryCapacity was 88.45 kWh.
- No DTC list and no indicators dict were present; their shapes are still unseen (parsed best-effort).
- Units: odometer km, speed m/s (CarTrak converts), tire kPa, temperatures C.

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
