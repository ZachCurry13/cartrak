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
1. User updates TrueNAS (Stop/Start) and confirms the app shows v0.2.0; then Ford live test (needs Application-Id + redirect URI).
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
