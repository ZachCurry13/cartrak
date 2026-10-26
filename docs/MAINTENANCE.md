# Maintenance, security and update protocol (reusable)

How CarTrak is kept patched without surprises. The reusable version of this protocol lives outside the repo; this copy is filled in for CarTrak. A key-free weekly audit runs from `.github/workflows/audit.yml` (keeping one open "Weekly audit" issue), next to the `security` workflow (govulncheck and CodeQL).

## Project
- **What it is / how to build and test:** `go build ./...`, then `go test ./...`, `GOOS=linux go vet ./...`, `gofmt -l .` (must print nothing) and the JS syntax check from CLAUDE.md.
- **Package managers in use:** Go modules and Docker. The web app has no npm packages.
- **Components that can be updated on their own:** the Go toolchain (the `go` line in `go.mod`, which CI and Docker read), Go modules, the pinned `cloudflare/cloudflared` image and the Go build image in the Dockerfile, and the GitHub Actions versions.
- **Known test failures to ignore:** none.
- **Release rule:** only with the user's OK, every time, following the release flow in CLAUDE.md. Patches wait on a branch.

## 1. Modular components and isolated updates
List what can move independently of the app itself and check each against its upstream: dependencies, pinned images, vendored files, plugins. An update can ship on its own when it touches only that component and changes no interface the rest of the app relies on. Anything that changes an interface is written up for the owner, not applied.

## 2. Auto-patch boundary
Apply, on a branch and without asking, only **patch or minor releases that fix a known vulnerability**. Everything else (feature releases, major versions, base-image or runtime jumps, anything that changes behaviour) is reported with a plain explanation of what changed and why it matters. Never merge to the main branch or release; that is the owner's call.

## 3. Security audit
Scan the code and its dependencies for known vulnerabilities and propose a targeted patch for each finding that applies, highest severity first. Findings in the language's standard library depend on the toolchain version doing the scanning, so check which version the real build uses before treating them as real.

| Stack | Vulnerability scan | Newer dependency versions |
|---|---|---|
| Go | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` | `go list -m -u all` |
| Node | `npm audit` | `npm outdated` |
| Python | `pip-audit` | `pip list --outdated` |
| Docker | compare pinned tags with the upstream latest release | same |

## 4. Feedback from GitHub
Review open issues, pull requests and bug reports for the repo and the components it bundles. Turn what is actionable into small, focused changes; list the rest.

## 5. Verify, and undo only your own change
After any update run the build and the full test suite. If something that passed before now fails, undo **only what this run changed** (for example `git restore go.mod go.sum`, or the specific files), never other work, and report why. Do not push a failing branch.

## 6. Headless and scheduled runs
- **Key-free weekly audit (recommended):** `.github/workflows/audit.yml` runs the scan and the version check every Monday and keeps one open issue up to date (closing it when everything is clear). It needs no API key and changes nothing.
- **Claude-driven runs:** `claude -p "run the maintenance protocol"`, or a scheduled workflow that does it and opens a draft PR. That needs an Anthropic API key stored as a repository secret and a workflow with write access, so set it up deliberately, not by default.

## Where results go
Keep a short log in `docs/STATUS.md` (3 to 5 lines per run: what changed, what needs a decision, what is next), and tell the owner in plain words about anything that needs one.

