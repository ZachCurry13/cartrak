# Status

- Released: v0.1.0 to v0.9.0 (latest tag v0.9.0, image :latest, linux/amd64+arm64). v0.9.0 (2026-10-13) contains the Status screen, real-data dashboard cards, phone tracking, the Vehicle health tab with vehicle type detection (admin setting, Ford garage engineType, reported metrics), maps (server-side OpenStreetMap tile proxy), the mileage log, and the held v0.8.2 Ford request-limit fix (no separate 0.8.2 tag was made).
- In progress (feature/v0.10): Web Push alerts (charge complete, plug in/out, low tire, low 12 V, unlocked, trip start/end). Optional small adds: read-only departure times / charge schedules card.
- Not yet checked against the real services: OpenStreetMap tiles and Nominatim (tested with local fakes), real trouble-code (DTC) data (the user's car sent none), the phone automations on a real phone. Ask the user to update TrueNAS (Stop, then Start) and try: Health tab, Maps, More > Mileage log, More > Phone tracking.
- Known: an earlier pushed commit on feature/v0.9 (now in main history) holds a real odometer value in internal/ford/testdata/bev_response.json history; the user chose to leave it (no VIN or location).
- Answered for the user: the Ford API in use (FordConnect 2.0 Query) is read-only (7 GETs); commands exist only in FordConnect v1 (partner Application-Id) or the unofficial FordPass API; recommended not to build control now.
