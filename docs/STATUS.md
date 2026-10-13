# Status

- Released: v0.1.0 to v0.8.1 (latest tag v0.8.1, image :latest, linux/amd64+arm64). v0.8.1 is the real Ford sign-in fix: the link now starts at Ford's real address (fcon-public/v1/auth/init, 16-character state, no scope), found by reading the open-source ha-fordconnect-query integration; the parser understands the real telemetry layout. v0.8.0 added the grouped More menu and the under-200-line file split.
- In progress (feature/v0.9): phone tracking (ingest API with device tokens, phone GPS points and start/stop events merged into trips, Ford fast-poll on start, Android Tasker/MacroDroid + iPhone Shortcuts recipes, Live-view GPS logger).
- Waiting on the user: retry Connect Ford on v0.8.1 (Ford pages: sign in, choose vehicle, Consent), then Admin settings -> Ford diagnostics -> Copy responses so the dashboard cards can be checked against the real data.
