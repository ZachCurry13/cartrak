# Status

- Released: v0.1.0 to v0.8.1 (latest tag v0.8.1, image :latest, linux/amd64+arm64). v0.8.2 (Ford rate-limit handling) is built on feature/v0.8.2 and merged into feature/v0.9, NOT released (the user said "Not yet").
- In progress (feature/v0.9, unreleased, CHANGELOG `## 0.9.0` drafted): the Status screen (`/api/health`, More -> Status, Home strip) and real-data dashboard cards (Doors and locks, Health; tires and locks parsed from the real list layout in `internal/ford/parse_lists.go`; the demo car uses the same layout). Browser-checked at desktop and a 328 px box; Go and JS checks pass.
- Next in 0.9: phone tracking foundation (ingest API with device tokens, phone GPS points and start/stop merged into trips, Ford fast-poll on start, Android Tasker/MacroDroid + iPhone Shortcuts recipes, Live-view GPS logger), then alerts via Web Push. Ask before releasing anything.
- Waiting on the user: Admin settings -> Ford diagnostics -> Copy responses after the 429 clears (needed to confirm real alerts, wallbox and schedule shapes); whether to release v0.8.2 on its own or together with 0.9.0.
