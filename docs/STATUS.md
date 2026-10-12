# Status

- Released v0.1.0 and v0.2.0 (tags v0.1.0, v0.2.0; image :0.2.0 and :latest, linux/amd64+arm64, public).
- Released v0.3.1 (tag v0.3.1, image :0.3.1 and :latest). v0.3.0 added Admin settings, the form-style TrueNAS guide and trust-proxy/session settings; v0.3.1 fixed the Connect Ford link path (/common/login/).
- Released v0.4.0 + v0.5.0 together as tag v0.5.0 (image :0.5.0 and :latest): Live tab/SSE, Road trip mode, per-device Data use, responsive layouts, gzip/ETag, FordConnect 2.0 provider, Ford diagnostics.
- Released v0.5.1. In progress (feature/v0.6, UNRELEASED, tests pass, not yet CI-checked): in-app Cloudflare tunnel (internal/tunnel, Remote access page, cloudflared bundled in the Docker image + CI docker smoke job), fragment-aware Ford callback page. Docs for the tunnel (docs/cloudflare.md, truenas.md, README) still need updating.
- Next: user updates TrueNAS and runs Connect Ford + Ford diagnostics; use the copied redacted responses to build proper dashboard cards for the real telemetry shape.
- Open: Ford endpoints unverified live; Docker image builds only in CI.
