# Road to 1.0

1.0 means the app can be left alone and trusted. The gate is correct data, not more features. No new features until the gate is met; releases in the meantime are fixes and checks (0.57 to 0.59).

## The gate
Cut 1.0 after about two weeks with no data-correctness fix, on every car.

## 1. Data you can trust
- [ ] Replay fixtures: each past trip-detection bug becomes a small synthetic replay test (split drive from a blank answer, stale position, odometer regression, late-reported drive, endless trip, missing start). Synthetic values only; never real positions, odometers or VINs from a user's export.
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
- [ ] Upgrade test: migrations run on a copy of a real database from an older version.
- [ ] Ford API change or outage: visible on Status, with an alert, no silent gaps.
- [ ] Documented limits (Ford gives live state only, so gaps cannot be recovered).

## 4. Scope decisions
- [ ] "All cars" combined view: finish or cut.
- [ ] Android shell: written down as out of scope.
- [ ] Shelved ideas (ChargePoint, AI, 2FA, comma 4, OBD-II) listed as not in 1.0.

## 5. Housekeeping
- [ ] Security and dependency audit per docs/MAINTENANCE.md.
- [ ] README and docs match the app; a short "what 1.0 promises" section (data format and exports stay stable).
