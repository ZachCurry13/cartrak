# Road to 1.0

1.0 means the app can be left alone and trusted. The gate is correct data, not more features. No new features until the gate is met; releases in the meantime are fixes and checks (0.57 to 0.59).

## The gate
Cut 1.0 after about two weeks with no data-correctness fix, on every car.

## 1. Data you can trust
- [ ] Replay fixtures: each past trip-detection bug becomes a small synthetic replay test. Started in 0.58.0 (internal/trips/replay_test.go and replay_scenarios_test.go: blank answer, odometer regression, quiet gap, two drives, sitting in the car, ignition blip); 0.59.0 added the missing start and stale position; 0.60.0 added the late-reported drive; still to add: endless trip. Synthetic values only; never real positions, odometers or VINs from a user's export.
- [x] A drive Ford reports only after it ended: a car first seen on and stopped, with the odometer 1 km or more past the last reading at a believable speed, now starts its trip at that last reading (0.60.0, `trip_late`). Still to confirm on a real drive with no signal.
- [ ] Fix a day verified on real data (Oct 22 to 24).
- [ ] Escape trip distance (8.0 mi against 2.5 mi on the odometer) explained.
- [ ] Trips that still draw as dashed straight lines explained (pending or implausible).
- [ ] A daily check of every car's trips for a stretch of weeks, with nothing to correct.

## 2. Checked on the real world
- [ ] Push alerts and phone automations on a real phone, and the notification badge.
- [ ] Real trouble-code (DTC) data.
- [ ] OpenStreetMap tiles.
- [ ] Electric demo after the fuel-wording changes.
- [ ] A second person's Ford account (data/fordtest/STEPS.txt).

## 3. Operations
- [ ] Restore test: back up, wipe, restore, everything comes back.
- [x] Upgrade test: every migration upgrades a database that lacks its column, with data kept (0.59.0, internal/store/migrate_test.go). Still worth doing once: open a copy of the real database from an older version.
- [ ] Ford API change or outage: visible on Status, with an alert, no silent gaps.
- [ ] Documented limits (Ford gives live state only, so gaps cannot be recovered).

## 4. Scope decisions
- [ ] "All cars" combined view: finish or cut.
- [ ] Android shell: written down as out of scope.
- [ ] Shelved ideas (ChargePoint, AI, 2FA, comma 4, OBD-II) listed as not in 1.0.

## 5. Housekeeping
- [x] Security and dependency audit per docs/MAINTENANCE.md: run 2026-10-27, govulncheck found nothing that affects CarTrak (GO-2026-5932, golang.org/x/crypto/openpgp, unmaintained, not used, no fix available). Repeat before 1.0.
- [ ] README and docs match the app; a short "what 1.0 promises" section (data format and exports stay stable).
