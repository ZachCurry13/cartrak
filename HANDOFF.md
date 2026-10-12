# Handoff

## State
Branch `feature/v0.1` (pushed) holds the whole v0.1.0 app; `main` has only the scaffold. No release yet.
Public repo: https://github.com/ZachCurry13/cartrak (MIT). Rules are in CLAUDE.md.

## Decisions
- Standalone Go app, SQLite (modernc, no CGO), embedded vanilla-JS PWA, admin + viewer roles.
- Ford: FordConnect v1 (owner link login, refresh token stored in DB). FordPass-style fallback deferred.
- Trips auto-classified personal/work by rules (days + time window, optional geofence); manual always wins.
- Ford-inspired palette only (no logos, fonts, or Ford name in branding); disclaimer in More page.
- Deploy: GHCR image via docker.yml, TrueNAS compose YAML with a cloudflared sidecar.
- comma 4 + StarPilot: planned as a separate data source; the device pushes drive summaries/GPS to CarTrak over the tunnel with a token (most private; Ford never involved). Ideas: openpilot engagement % per trip, disengagements, efficiency openpilot vs manual, device health.

## Next immediate steps
1. User reviews; release v0.1.0 only on their OK (release flow in CLAUDE.md).
2. Verify against real Ford: token URL, link URL, api-version, field names (see internal/ford/fordconnect.go, parse.go).
3. Build the comma push endpoint + device script; add FordPass fallback if wanted.

## Known gaps
- Ford endpoints unverified live. Docker image untested locally (no Docker); CI builds it.
- Local-only files (git-excluded): `.secrets/` (Ford creds + data dictionary PDF), `.claude/`.
- Demo launch config: `.claude/launch.json` (git-excluded).
