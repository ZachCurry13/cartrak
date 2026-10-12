# CarTrak

Self-hosted Mustang Mach-E GT Performance tracker. Go backend, SQLite, embedded
mobile-first PWA. Runs as a TrueNAS custom app behind a Cloudflare tunnel.

## Rules (adapted from the family-projects working rules)
- **Plan first.** Non-trivial or multi-file changes are agreed before editing; offer options with a recommendation.
- **Releases only with the user's OK**, every time. "Build it" never implies a release.
- Work on `feature/vX.Y` branches. Features bump minor (0.2.0), fixes bump patch.
- **Release flow** (after OK, one command per call): `git checkout main` -> `git pull --ff-only` -> `git merge --ff-only feature/vX.Y` -> `git push origin main` -> `gh workflow run docker.yml --ref main -f version=X.Y.Z` -> `gh run watch <id printed by that command>` -> confirm the GitHub Release -> update HANDOFF.md. Push to main publishes `:main`; a release publishes `:X.Y.Z` and `:latest`.
- Every user-facing change ships with: a plain-English CHANGELOG entry (emoji headings), README/docs in sync, a new service-worker cache name in `internal/web/static/sw.js` (plus new files added to its list), and updated `docs/STATUS.md` and `HANDOFF.md` after a release.
- Keep files small (~200 lines); split by responsibility. Read only what a task needs.
- Delegation: `explorer` for finding code, `worker` for low-risk edits. Architecture, tricky bugs, and anything touching drives are done directly.

## Privacy (the repo is PUBLIC)
- Never commit the user's setup: hardware, LAN IPs, ports, pool names, time zone, names, VIN, location, Ford credentials.
- Neutral wording: no personal names or pronouns in the app or docs.
- Local-only files go in `.git/info/exclude`, not `.gitignore` (`.secrets/`, `Ford API.txt`, `.env*`, `data/`, `.claude/settings.local.json`, `devfixture.js`).
- Secrets are settings/env vars: never sent to the browser, never logged.
- Never sign in or create accounts on outside sites in the browser; the user does that. Testing the local app with test accounts is fine (passwords stay in the scratchpad).

## Checks before calling work done
- Go: `go test ./...`, `GOOS=linux go vet ./...`, `gofmt -l .` clean. Every outside service (Ford, Cloudflare) gets an `httptest` fake.
- JS: `for f in internal/web/static/js/*.js; do node --input-type=module --check < $f || echo BAD $f; done`
- UI: check at 360x760 with awkward data (long names, many trips), measure `scrollWidth`, tap targets, light and dark themes. Use a local-only `devfixture.js` harness. If the browser pane is hidden, measure with scripts, not screenshots.
- Flex rows with buttons give text a `basis-*` so buttons wrap on phones.

## Windows editing traps
- Prefer the Edit tool. Scripted edits go in script files in the scratchpad, not inline `node -e`. No Python on this machine.
- `grep -rnF '() =>' --include=*.md .` after scripted doc edits.
- Clones use LF endings (`core.autocrlf=false`).

## Users
Admin (manages everything) and viewer (read-only). No kid accounts or age groups.
