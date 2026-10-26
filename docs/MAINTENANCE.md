# Maintenance, security and update protocol

How CarTrak is kept patched. Used by hand, by `claude -p`, or by a scheduled run.

## What can be updated on its own
| Part | Where it is pinned | Check |
|---|---|---|
| Go toolchain | `go` line in `go.mod` (CI and Docker read it) | `govulncheck` (security workflow) |
| Go modules | `go.mod` / `go.sum` | `go list -m -u all`, `govulncheck` |
| Cloudflare connector | `FROM cloudflare/cloudflared:<tag>` in `Dockerfile` | latest tag of cloudflare/cloudflared |
| Docker base images | `Dockerfile` | newer tag of the same major |
| GitHub Actions | `.github/workflows/*.yml` | newer major/minor of each action |

Each is a pin, so one can change without touching app code. A bump is still a rebuild of the image, so it ships in a release.

## Safe to patch without asking
- Patch releases of the Go toolchain and Go modules, and cloudflared patch tags.
- Only after `go build ./...`, `GOOS=linux go vet ./...`, `gofmt -l .`, `go test ./...` and the JS syntax check all pass.
- Anything bigger (major versions, changed behavior, new config) is reported to the user, not applied.

## Routine
1. `govulncheck ./...` (the weekly security workflow does this). Fix high/critical findings first.
2. `go list -m -u all`, newest cloudflared tag, newer action versions.
3. `gh issue list` and `gh pr list` for outside reports; turn real ones into focused changes.
4. Apply safe patches on the current feature branch (or a `fix/` branch), run the checks above.
5. If a check fails, undo only your own change (`git checkout -- <files>`) and report why.
6. Report: what changed, what needs a decision.

## Guardrails
- **Never release, merge to main or auto-merge without the user's OK.** Patches are committed to a branch.
- Never touch the user's data, settings or deployment. Never put personal details in commits.
- Rollback means reverting your own uncommitted or just-made change, nothing else.

## Headless
```bash
claude -p "Follow docs/MAINTENANCE.md: audit, patch what is safe on a branch, report."
```
`.github/workflows/security.yml` already runs `govulncheck` weekly and on pushes.
