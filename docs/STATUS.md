# Status

- Released v0.1.0 and v0.2.0 (tags v0.1.0, v0.2.0; image :0.2.0 and :latest, linux/amd64+arm64, public).
- Released v0.3.1 (tag v0.3.1, image :0.3.1 and :latest). v0.3.0 added Admin settings, the form-style TrueNAS guide and trust-proxy/session settings; v0.3.1 fixed the Connect Ford link path (/common/login/).
- Released v0.4.0 + v0.5.0 together as tag v0.5.0 (image :0.5.0 and :latest): Live tab/SSE, Road trip mode, per-device Data use, responsive layouts, gzip/ETag, FordConnect 2.0 provider, Ford diagnostics.
- Released v0.6.0 (tag v0.6.0, image :0.6.0 and :latest): in-app Cloudflare tunnel (Remote access page, cloudflared bundled), fragment-aware Ford callback page, remote access before Ford in Getting started, docs.
- Released v0.7.0 (per-trip time zones, local trip times, offline last readings). feature/v0.7.1 (unreleased): accept Ford returning to the site root; no-path redirect buttons.
- feature/v0.7.2 (unreleased): Connect Ford page shows the redirect in use with one-tap choices. feature/v0.8 has the file-split refactor (all files under 200 lines) and is next for phone tracking.
- In progress (feature/v0.7, unreleased): per-trip time zones from location (internal/tzlookup, trips.tz + migration), trip times shown in local time with a per-device switch, offline last-readings cache. Binary grew ~15 MB for the embedded zone map.
- Next: user updates TrueNAS and runs Connect Ford + Ford diagnostics; use the copied redacted responses to build proper dashboard cards for the real telemetry shape.
- Open: Ford endpoints unverified live; Docker image builds only in CI.
