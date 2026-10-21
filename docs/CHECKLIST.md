# CarTrak checklist

What we work through together to make CarTrak better. Tick items as they are done. Top three first.

## Top three
1. [ ] **Two debug reports.** One after a real drive in the hybrid (with a fill-up if possible), so we see what Ford sends while it moves. One covering the day of the doubtful efficiency record (5 days or more) on the electric car.
2. [ ] **Screenshots of what feels off.** Home, Trips, Insights, More and Live, at phone width, in light and dark, with a sentence on what bothers you.
3. [ ] **Consistency pass.** One set of spacing, card, heading, button and empty-state rules on every page, the same words for the same things.

## Data to pull (you)
- [ ] Debug report after a drive in the hybrid (electric distance counter, trip fuel economy, fuel level)
- [ ] Debug report covering the day of the 6.27 mi/kWh record (the trip energy check)
- [ ] Car-fields export (7 days) for each car, to compare which fields CarTrak reads and which it ignores
- [ ] A debug report from a plug-in hybrid, if one is available

## Screenshots and feedback (you)
- [ ] The screens you open most, and the ones you never open (what to simplify or hide)
- [ ] Anything cramped, inconsistent or "not like a real app", circled or described
- [ ] Who the app is for (just you, family, others), so we know how far to simplify
- [ ] The one thing you would show a friend: we polish that first

## Look and usability (me)
- [ ] Consistency pass: spacing, cards, headings, buttons, empty states, wording
- [ ] Mixed fleet walkthrough: every page as an electric owner, a hybrid owner, a viewer and an admin
- [ ] Plug-in hybrid trip page shows both kWh and gallons (today fuel replaces the battery stats)
- [ ] Dark and light check of every screen at 360 px

## Use the data better (me)
- [ ] Electric share of each hybrid trip, if Ford sends the electric distance
- [ ] Trip energy checked against the battery percent drop (stops a stale reading from skewing a trip)
- [ ] Hybrid and gas: economy against temperature and speed, 12 V battery trend
- [ ] Cost per mile for every car, side by side in All cars
- [ ] Fuel columns in the trips CSV and reports
- [ ] Charging loss from real receipts: measured loss by charger type (fast, public AC, home) with how many receipts back it, suggestions on the setting, home cost from the wall kWh, odd sessions flagged (cold battery or a charger fault), and loss against outside temperature
- [ ] Home charging loss: enter the wall kWh from the home charger or meter for a few sessions, so home is measured on its own

## More debug (me)
- [ ] Alert decisions and skipped alerts, with why
- [ ] Readings dropped because they came from another vehicle
- [ ] Trip energy check in the debug report
- [ ] Recent setting changes (no secrets)
- [x] Trips say when their distance was corrected and what the odometer said; manual trip changes are logged (done in 0.53)
- [x] Slowest requests (done in 0.53)
- [ ] Short and full report sizes

## Later
- [ ] Lossless compression of the stored raw readings (only when asked; raw data is kept for at least a year)
- [ ] Backup check: open a backup and count what it holds
- [ ] Quick "mark this week personal" from the trip list
