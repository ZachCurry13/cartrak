# Status

- Released: v0.1.0 to v0.7.2 (latest tag v0.7.2, image :latest, linux/amd64+arm64). Highlights: Ford provider (FordConnect 2.0), trips/charging/rules, Live view + Road trip mode + Data saver, in-app Cloudflare tunnel, per-trip time zones, offline last readings, Connect Ford page with one-tap redirect choices.
- In progress (feature/v0.8): source and test files split so everything is under 200 lines (done, no behaviour change). Next on this branch: phone tracking foundation (ingest API with device tokens, phone GPS points and start/stop events merged into trips, Ford fast-poll on start), Live-view GPS logger, Android Tasker/MacroDroid and iPhone Shortcuts recipes.
- Waiting on the user: link Ford and run Admin settings -> Ford diagnostics -> Copy responses, so dashboard cards can be built from the real telemetry shape.
- Open: Ford sign-in/response shapes unverified live; Docker image builds are verified in CI.
