# Changelog

## 0.21.0

### 🏠 A clearer Home
- Under the battery ring, small chips say at a glance whether the car is **Driving, Charging or Parked**, whether it is **plugged in**, and whether it is **locked**.
- A **Needs attention** card (only when there is something) lists service reminders that are due or coming up and trips waiting for a Work or Personal label, each one tap away.
- **Last 7 days** and **Last trip** (with its named places) now sit right under the battery instead of at the bottom.

### 📤 More in the CSV exports
- **Charging** has an **Export CSV** button: every session with the energy added, the charger's own number and efficiency where entered, where it was, peak power and cost.
- The trips CSV now also has energy used, whether it is an estimate, and the named start and end places.

### 👁️ Easier to read in light mode
- A contrast check of every text and button colour found that a few in light mode (links, green buttons and chips, amber and red labels) were slightly under the recommended contrast. They are a little deeper now. Dark mode already passed.

### 💽 Disk space
- **Status** now shows how much room is left on the disk CarTrak stores its data on (on the Docker image), warning when it is getting low and going red when it is nearly full, with a link to Free up space. A new optional alert, **Low disk space**, warns you at most once every three days. CarTrak adds readings all the time, so a full disk would stop everything.
- The **Car readings** line on Status now tells apart Ford answering from the car itself being asleep ("Ford is answering, but the car itself last reported 6 h ago. It is probably parked and asleep, which is normal.").

### 🔔 See what alerts were sent, and a guide to everything
- **More, Notifications** now ends with **Recent alerts** (admins): every alert CarTrak decided to send, and whether it reached a device, none had it turned on, or it was held back by quiet hours. If an alert you expected is not listed, CarTrak never decided to send it.
- **More, What CarTrak can do** lists every feature in one place with a line about each and a link, so nothing gets forgotten.

### 🚨 Parked watch (experimental)
- **More, Parked watch** uses the car's motion sensors to notice a bump on a parked car (another car's door, for example), and can alert you with **Bump while parked** and **Car alarm** (both off until you turn them on under More, Notifications). The bump level is a setting. Because the car only reports now and then, a short knock can be missed, so the page shows how many parked readings were checked, the strongest motion seen and what it noticed, to help you judge how well it works. See docs/places-and-maintenance.md.

### 📍 Trip rules can use your named places
- When you make a trip rule that applies only near a place, you can now pick one of the places you named under More, Places instead of typing coordinates.

### 🧾 Mileage log shortcuts
- **This week, Last week, This month** and **Last month** buttons build the log for that period in one tap.
- When a period covers more than one calendar week, the log also lists the miles for **each calendar week**, to check against a weekly sheet.

### ❄️ Range that knows about the weather
- The 20 to 80% line on Home now adds **"At today's 28°F: about 96 mi"** when today's temperature makes a noticeable difference from your recent average. It uses how far you really got on trips in that kind of weather (below freezing, 0 to 10, 10 to 20, and 20 °C and above).
- The **road-trip planner** has a **Weather** choice (today's temperature, your recent average, or any weather group), so you can plan a winter trip with winter range.
- **Trips** shows the range for each weather group next to the monthly table.

## 0.20.0

### 📺 Live fits one screen and follows what the car is doing
- The Live screen no longer needs scrolling. It shows the battery ring and six tiles that change with the car: **driving** (speed, this trip, trip time, miles to your reserve, range, driving today), **charging** (power, time to your target, energy added this session, range, energy left, outside temperature) or **parked** (range, energy left, doors, driving today, outside temperature, odometer). A small chip says which it is. The phone GPS tools moved into a collapsed section at the bottom.

### 🕐 Know when the car last reported
- The top of every page said "updated 6 minutes ago" even when that was just CarTrak asking Ford (for example after a server restart). It now says **car reported …**, using Ford's own update times, so it tells you when the car itself last sent something. A sleeping car simply shows an older time. Hover or press and hold it to see when CarTrak last checked Ford, and it says so if Ford is not answering. Live shows the same.

### 🧹 Free up space (optional)
- **More, Backup, Free up space** can remove the full raw Ford detail of older readings (you choose how many recent days to keep, and type TRIM to confirm) while keeping trips, charging, charts and each reading's main numbers. It works in small background steps, and the freed space is reused so the database stops growing for a while. See docs/backup.md.

### 🏠 Home: the last 7 days
- Home has a **Last 7 days** card: distance, trips, energy used and efficiency (tap it to change the unit).

### ♻️ Restore a backup from the app
- **More, Backup, Restore a backup:** upload a backup (or pick one you copied into the data folder's restore folder), see what is in it, type RESTORE, and CarTrak restarts with it. Your current data is kept beside it as `cartrak.db.before-restore-<time>`, so nothing is lost. See docs/backup.md.

### 🕒 Trip time and time driving today
- **Live** has two new tiles: **Trip time** (how long the current trip has been going, and when it started) and **Driving today** (all the time spent driving since midnight, including the trip in progress). Both keep counting between readings.

### ⏱️ Time: parked, driving and charging
- **More, Time** shows what share of the time the car was parked, driving or charging over the last 7 days, 30 days or year, a bar for each day, and where it was parked (Home, the office and other places you have named, or Elsewhere). It uses the trips and charging sessions CarTrak already recorded.

### 🗺️ Open the car in your map app
- The map on Home is now a link: tap it to open the car's spot in Google Maps. There are also **Open in Google Maps** and **Directions to the car** buttons (and **Open in Apple Maps** on iPhone), and the trip page has the same for where it ended. Nothing is sent to Google until you tap.

### 🔁 Tap an efficiency figure to change its unit
- Tap any efficiency number (on Trips, a trip, a journey or Efficiency insights) to switch all of them between miles per kWh, watt-hours per mile and miles per percent of battery. Metric devices get kWh per 100 km, Wh per km and km per percent. CarTrak remembers your choice.

### 🔌 Measure your charging loss
- On each charging session, admins can type in **kWh from the charger** (from your charger's app). CarTrak shows how much of it reached the battery, and after three sessions it uses your own measured average instead of a fixed 90% when it works out what charging cost.
- A new setting, **Slow-charging efficiency**, sets the number used until it has been measured.

## 0.19.0

### 🧳 Journeys
- On **Trips**, admins can select several trips and **combine** them into a journey (a road trip, or a work day with several stops). The journey page shows total distance, driving and elapsed time, the stops between legs (and how much you charged), energy used and efficiency, what charging cost, and a map of every leg. Trips stay as they are and can be removed from a journey at any time. See docs/road-trips-and-charging.md.

### 🔗 Share a trip or journey
- Admins can make a **public link** to a trip or a journey from its **Share** card. You choose the route (with the first and last quarter mile left out, the whole route, or numbers only), whether to show city names and the date and time, and when the link expires. The page needs no sign-in and shows only that, never your VIN, live position or other trips. The link is shown once; **More, Security, Shared links** lists every link and turns any of them off at once. See docs/sharing.md.

### 📍 Places, trip review and maintenance
- **More, Places:** name Home, the office and other places (from where the car is, or from a trip's start or end). Trips then read "Home → Office", and you can get an alert when the car arrives at or leaves a place.
- **Trips to review:** a banner on Trips and a quick screen to tap Work or Personal on trips that no rule could label.
- **More, Maintenance:** service reminders by distance, by time or both (tire rotation, cabin filter, brake fluid and so on), with an alert when one comes due and a Done button that starts the next interval.
- **Charging stopped early:** a new alert (off until you turn it on) for a charge that stops while plugged in at a lower level than your sessions usually end at. See docs/places-and-maintenance.md.

### 📈 Efficiency insights and a weekly summary
- **More, Efficiency insights:** your miles per kWh by outside temperature and by month, and your most and least efficient trips, so you can see what the cold really costs.
- **Weekly summary:** an optional Sunday evening notification with the week's distance, trips, charging and efficiency (turn it on under More, Notifications).

### 🧭 Road-trip planner
- **More, Road-trip planner:** enter a trip distance, your starting charge and your average charging power. It uses how far your battery really goes per percent (from your own driving), your reserve and your charge target, and says how many charging stops the trip needs, about how far in each one is, what to charge to, and about how long. It is a guide, not a route.

### 🏠 Home Assistant
- A read-only address for smart-home tools (`/api/ha/state`, read with a phone-tracking device token) gives the battery, range, plugged in, charging, locked and a few more values, with no location or VIN. See docs/home-assistant.md for sensor examples.

### ☁️ Update the Cloudflare connector from the app
- When Cloudflare publishes a newer connector, **More, Remote access** says so and has an **Update** button. It downloads the file from GitHub, checks it against the checksum GitHub publishes, runs it once, switches the tunnel over, and puts the old connector back on its own if the tunnel does not reconnect within 90 seconds.
- **More, About** now lists the versions of CarTrak, the connector, SQLite and Go, and which of them have an update.
- The repository gets a weekly Dependabot check for newer libraries and base images.

## 0.18.0

### 🅿️ Parked drain
- The Health page has a new **Parked drain** section for electric cars: how much charge the car loses while it sits (not driving, not charging) over the last 7 and 30 days, as a percent a day and in kWh, with a note on whether that is normal and a list of the recent parked stays. It needs about half a day of parked readings.

### 🔋 Charging history and cost
- The **Charging** tab now shows the energy each session added (in kWh), and a **Charging cost** card with every month: sessions, kWh, cost, miles driven and **cost per mile**.
- Set your price per kWh at home (and away from home, and a currency symbol) in More, Settings. Mark one session as **Home** and CarTrak recognises home from then on. Slow chargers include about 10% lost as heat.

### 💾 Backup
- **More, Backup** downloads a copy of your data: a small backup without the raw Ford readings, or a full one. Sign-in sessions are never included. See docs/backup.md for restoring.

### 🔐 Security
- **More, Security** shows who is signed in right now, with a button to sign a device out, and a **sign-in log** of the last 100 attempts (name, success or failure, device, address). Passwords are never stored.

### 📱 Phone shortcuts
- On Android, press and hold the CarTrak icon to jump straight to Live, Charging, Trips or Health.

## 0.15.0

### 🔌 Plug in tonight?
- A new alert, off until you turn it on under More, Notifications: one reminder in the evening (9 pm by default) if the car is parked, not plugged in and the battery is below 60 percent. An admin can change the time and the level in Settings. It is sent at most once a day, and only if the car has reported in the last two hours.

### 🧾 Mileage log rounding
- The mileage log now rounds **each trip up to the next whole mile** before adding the day, so 0.1 mi counts as 1 and 26.2 as 27. You can switch to the old way (add the day, then round to the nearest mile) with the new **Miles** choice on the Mileage log page.

## 0.14.0

### 🎨 A new look
- CarTrak now has a darker, calmer look with one green accent. Light mode follows the same design with deeper colours, and it still follows your phone or your choice under More, Display and data.
- **Home** shows the battery as a ring with the percentage and range inside, then Energy left and Battery temp, your reserve and the 20 to 80% figure. Charging and Vehicle statuses now read as plain words (for example Not ready, Disconnected, Park) instead of Ford's capital letters.
- **Live** uses the same ring on its big Charge tile. The ring turns amber at 20% or less and red at 10% or less.
- The bottom bar has an icon above each label, and on a wide screen the left sidebar has them too.
- Cards, buttons, chips, the Trips filters, the More menu and the Health and Data pages share one set of colours, sizes and corners. The day chart on Data follows the theme.
- Nothing about what CarTrak records or how it works has changed.

## 0.13.0

### ✅ Mileage check
- The **Mileage log** now has a **Does it add up?** card. It compares the car's odometer, which is exact, with the trips CarTrak recorded for the same weeks or dates, and says **Matches** or how many miles are missing.
- Every stretch of driving that no trip covers is listed with its time window and where the car was. **Add as work trip** or **Add as personal trip** fills it in from the odometer in one tap. When the miles sit right before or after a recorded trip (it was noticed or closed a little late), **Add to that trip** widens that trip instead.
- Admins only. See docs/mileage.md.

## 0.12.0

### 🔎 Car data, explained
- The **Car data** page now sorts everything the car reports into tabs: Summary, Battery and charging, Driving, Body and security, Settings and schedules, and Car. Every value has a plain-English name, and a **?** next to it explains what it means and shows Ford's own name for it. **All fields** still lists every raw value, with search.
- **Look back in time.** Pick any day and time (or step Earlier and Later) to see the car as it was then. A chart of the day shows the battery, with bands for driving and charging; tap it to jump to that moment.
- Values that Ford updates only when the car wakes now say how old they are.

### 🧭 How far between two charge levels
- Home now shows how far a stretch of battery goes, for example **20 to 80%: 142 mi**, worked out from how you have really been driving lately (so it follows the seasons), not just from the car's guess. The low end is your road-trip reserve and the high end is your charging target.
- **Trips** has a **Range by month** table, so you can see the winter drop and the summer recovery. It fills in as trips build up.

### 🔋 Calmer 12 V battery warnings
- A parked car often shows a middling 12 V reading (around 12 V and 50 to 70 percent) until it wakes and recharges, so the Health page now explains that instead of flagging it. Real problems (below 11.9 V or 50 percent) are still flagged.
- If a shop has tested the battery and it is fine, an admin can tap **It was tested and is fine** to pause its warnings and alert for 90 days.
- Clearer wording on the main battery's hidden top and bottom buffer, and a note that the 12 V percentage is the car's own estimate.

## 0.11.0

### ⚡ Road trips and charging
- **Energy left** is now on the Live screen (kWh, and about how much the battery holds when full).
- **Miles until your reserve:** set the level you do not want to go below (default 20%) under More, Display and data. Home and Live show how far you can go before the battery reaches it.
- **Time until a charge level:** while charging, pick 80%, 90%, 100% or any percent and see how long it will take and around what time. For 100% it uses the car's own figure; otherwise it is an estimate from the charging power, allowing for fast chargers slowing down above 80%. On Live, tap the tile to switch between 80, 90 and 100.
- **Trip efficiency:** every trip shows the energy it used and its miles per kWh (or kWh per 100 km if your units are metric), in the trip, in the trips list and as a total for the period. Trips from before this update are estimated from the battery percentage and marked as approximate. While a trip is in progress, Live shows its efficiency so far.

### 📍 GPS in the background
- CarTrak now accepts the **OwnTracks** app (iPhone and Android) and the docs have step-by-step settings for **GPSLogger** (Android), so your phone can send its position while the app is closed, with nothing to keep open. The phone sign-in also accepts a user name and password (the token is the password), which these apps use. See docs/phone-tracking.md.

## 0.10.1

### 🛠 Health tab fix
- Fixed the Health tab seeming to do nothing on a phone. Every reading carries a large block of raw data, and on an install that had been running for a few days the charts read through all of it, which was slow enough to stall the whole app. The charted values now live in a small table of their own: new readings add to it as they arrive, and readings from before the update are copied across quietly in the background (the charts fill in as it goes, and the app stays responsive).
- The Health page now says "Loading vehicle health…" while it works, and shows a clear message with a Try again button if something goes wrong, so it never looks dead.
- Trip routes, the trip distance check and the Trips map read only positions and speed instead of every field, so they open faster too.

## 0.10.0

### 🔔 Alerts on your phone
- New More, Notifications: turn alerts on for this phone or computer and get a notification for charging complete, plugged in, unplugged, a low tire, a low 12 V battery, the car left unlocked, a trip started or ended, and no readings from the car for hours. Each device chooses its own alerts and can set quiet hours, and a button sends a test alert.
- Each alert is sent once, and a restart does not repeat what was already true.
- Private: alerts are encrypted for your device, say things like "Charging complete" and never include a location. They go through your browser maker's push service, which can see that one was sent but not what it says. CarTrak only contacts real push services.
- Needs the secure address (for example your Cloudflare address). On iPhone, add CarTrak to the Home Screen first. See docs/notifications.md.
- Status shows an Alerts line.

## 0.9.1

### 🛠 When Ford's servers return an error
- If Ford answers a request with a server error such as HTTP 500, CarTrak now shows what Ford said alongside it (with any vehicle ID removed) instead of only the number, so the cause can be seen on the Status page and in the log.
- It is treated as Ford having trouble, not as a problem with your setup: Status says so in plain words, and CarTrak tries again after a minute, then 2, 4 and 8 minutes, never waiting more than 10. Before, it waited for the normal parked interval, which could be much longer.
- When Ford answers normally again the message clears by itself.

## 0.9.0

### 🩺 A Status screen that says what to do next
- A new Status page (More, then Status) checks the whole setup in plain language: Ford settings entered, Ford linked, last reading and how old it is, remote access, and whether an update is available. Each line says what is wrong and has a button to fix it.
- Home shows a short strip with the same result, so a problem is visible without opening any menu. Anyone signed in can see it; it never shows keys or passwords.

### 🚗 A dashboard built from the real car data
- New Doors and locks card: whether the car is locked and which doors, the tailgate or the hood are open.
- New Health card: warning lights the car reports and low tire pressure alerts, or "No warnings".
- Tire pressures and the lock state now read Ford's real layout (a list per wheel and per door), so the Tires card and the Locks line fill in with a linked car.
- The built-in demo car now uses the same layout, so these cards can be previewed without a car.

### 🧾 Mileage log
- New More, Mileage log: pick calendar weeks (for example 26-28) or dates and get a business mileage log in your sheet's layout: Date, Origin City, Destination City, Description, Total Miles Driven, plus Total Mileage Recorded. One row per day; the destination list is the city where you parked after each trip, each city once and in order, without your starting city.
- One-tap copy, tab-separated so it pastes cell by cell, with or without the headings and total.
- City names come from OpenStreetMap's Nominatim service, asked by your server only when you build a log, with locations rounded to about 100 m, and every answer is remembered. It can be switched off in Settings (then only remembered places are named). See docs/mileage.md.

### 🗺️ Maps
- Trip pages now show the route on a real map with start and end markers (your phone's GPS track when there is one). Home shows a small map of where the car is, and Trips has a new Map view with every trip in the chosen range, work in amber and personal in blue; tap a route to open the trip.
- Drag to move, + and − buttons, mouse wheel or two-finger pinch to zoom.
- Private by design: your phone or computer never contacts the map provider. CarTrak's server downloads the tiles from OpenStreetMap, keeps a copy on disk and passes them on, so repeat views are fast. Only signed-in people can load tiles, and only real map images are passed on.
- Settings: Maps can be switched off (plain route lines, no map requests), and a different tile server and credit line can be set. See docs/maps.md.

### 🩺 Vehicle health tab
- New Health tab: voltages, battery health, oil and fuel, tires, trouble codes, Ford alerts and warning lights, with a "Needs a look" list at the top and small daily trends for the 12 V battery, the main battery's estimated capacity and oil life.
- Built for each kind of car. CarTrak now knows whether the car is electric, plug-in hybrid, hybrid or gas: it reads Ford's vehicle record first, falls back to the data the car reports, and you can choose the type under Settings, Vehicle type. Oil life and fuel show for cars with an engine; the main battery shows for electric and hybrid cars. Battery cars never show oil, even though Ford lists an oil life of 100 percent for them.
- Home follows the type too: gas and regular hybrid cars have no Charging tab or card, a gas car has no Battery card, and cars with an engine get a Fuel and oil card.
- Each value says how old it is when it is more than two hours old (a parked car sends some values only when it wakes). The Home battery card says so as well.
- Fixed: Home showed the outside temperature as 32 °F on the real car because Ford's ambient temperature field reads 0; the real outside temperature is used now. Door names no longer say "Unknown".
- The demo car now uses the real data layout and can act as an electric, gas, hybrid or plug-in hybrid car (CARTRAK_DEMO_POWERTRAIN).

### 📱 Phone tracking (optional)
- New More, Phone tracking page: pair a phone, get a one-time token, and copy ready-made request examples. A test button confirms the phone can reach CarTrak, and the page lists the latest messages so you can see an automation fired.
- When a phone says a trip started, CarTrak asks Ford right away and checks more often until Ford reports the trip; when it says the trip ended, CarTrak checks again so the trip closes promptly. Ford still decides what counts as a trip, so odometer and battery numbers stay exact.
- GPS points from the phone are drawn on the trip page, a finer route than Ford's readings.
- Live page: Record GPS from this phone sends the phone's location while the page is open (with a queue so a dropped signal loses nothing).
- Step-by-step recipes for Tasker, MacroDroid and iPhone Shortcuts are in docs/phone-tracking.md.
- Tokens are stored only as a hash and can be removed any time. Optional setting: delete phone GPS points after a number of days (off by default; trips are never deleted).
- Status shows a Phone tracking line.

### ✨ Tidier Home and sidebar
- The tabs in the left sidebar on a computer or wide tablet were stretched to fill the whole column. They are now compact, normal-sized rows.
- The Getting started card on Home now only shows while Ford is not linked yet. Once you are set up it is gone; the checklist stays available under More, Getting started.

### 🧹 Under the hood
- The Ford tests were split into smaller files.

## 0.8.2

### 🐢 Playing by Ford's request limit
- Ford only allows about one request per endpoint every 30 seconds, and answers HTTP 429 ("too many requests") to anything faster. CarTrak was making a burst of six requests right after linking and your Test Ford connection added six more, so every endpoint was refused. CarTrak now never asks the same endpoint more than once per 30 seconds, reuses a recent answer instead of asking again, adds only one of the slower endpoints per reading, and after a 429 stops sending anything and waits (30 s, then 1 min, 2 min, up to 10 min).
- Road trip mode now means a reading every 30 seconds (it was 10, which Ford would refuse). All polling intervals have a 30 second minimum.
- Test Ford connection says "Rate limited" with how long to wait, instead of a plain red error.

## 0.8.1

### 🔗 Connecting Ford (the real fix)
- CarTrak now starts the sign-in at Ford's real sign-in address (the same one other FordConnect Query apps use), so Ford shows its own sign-in and vehicle-consent pages. Earlier versions started at a back-end address that always failed with "RESTApiCallForUserInfo".
- The sign-in link now carries a short `state` (Ford rejects a long one) and no extra scope fields.
- The token step tries Ford's documented form and, if Ford refuses it with a 400, again without the scope field.
- CarTrak now understands the real telemetry layout (battery, range, odometer, location, ignition, plug and more), so the dashboard fills in once Ford is linked.

## 0.8.0

### ✨ A cleaner More page
- More is now a short grouped menu (Connections, Trips, This device, Admin, Help) with status chips, instead of one long page of cards. Each row opens its own page: Ford connection, Remote access, Display and data, People, Settings, Getting started, About and updates.
- On tablets and computers the menu uses two columns.
- Slightly darker grey text in light mode for easier reading.

### 🧹 Under the hood
- Source files were split so none is over 200 lines; behaviour is unchanged.

## 0.7.3

### 🔗 Connecting Ford
- CarTrak now asks Ford which redirect address is registered on your app and uses one Ford accepts. Pressing Connect Ford checks first, so Ford is never sent an address it will reject (the repeating redirect_uri_mismatch error). The Connect Ford page shows what Ford said for each address.
- Clearer message when Ford accepts the address but reports its own sign-in problem.
- The Ford sign-in address setting accepts a link copied from Ford documentation, with {client_id}, {redirect_uri}, {state} and {scope} placeholders.

## 0.7.2

### 🔗 Connecting Ford
- The Connect Ford page now shows the exact address CarTrak asks Ford to return to, with one-tap choices: your site address, the address with /ford/callback, or http://localhost. Tap the one that matches what is registered on your Ford app. If the value is fixed by an environment variable or config.yaml, it says so.

## 0.7.1

### 🔗 Connecting Ford
- Ford apps registered with just the site address (no /ford/callback path) now work: CarTrak finishes the link when Ford returns to the site root.
- The Remote access and Admin pages have buttons for the address with and without the path, so it can match exactly what is registered on your Ford app.

## 0.7.0

### 🕒 Time zones that follow the car
- Each trip now uses the time zone where it **starts**, worked out from its location (no internet needed). "Weekdays 8am to 5pm" means 8 to 5 wherever you are driving, so a trip in New York is judged on New York time even if your server is in Minnesota.
- Trip times are shown in that local time with the zone name (for example "10:16 PM EDT") when it differs from your device. A per-device setting (More, Show trip times in) switches to your own device time.
- The time zone setting is now the **default**, used only when the car location is unknown. The CSV export gains local start time and time zone columns.

### 📴 Works without signal
- Home keeps the last readings on your device and shows them with an "offline" banner when there is no connection. Your server keeps collecting from Ford while your phone is offline, so nothing is lost; open the app again to see it all.

## 0.6.0

### ☁️ Remote access inside CarTrak
- A new Remote access page (More): paste your Cloudflare tunnel token and public address, switch it on, and CarTrak runs the connector for you, with a status and a connector log. The Docker image now includes cloudflared.

### 🔗 Connecting Ford
- Ford sending the code after a # in the address is now handled: the callback is a page that reads the whole address, and the paste box understands it too.
- Getting started now puts remote access before linking Ford, and the Connect Ford page warns when the redirect is a plain http address that Ford will refuse.

## 0.5.1

### 🔗 Connecting Ford
- A new **Connect Ford** page with two steps. If your Ford app was registered with the redirect `http://localhost`, you can now finish linking by pasting the address Ford sends you to. Ford apps registered with CarTrak's own address still connect automatically.
- Clear messages when Ford refuses: the Azure error is explained in plain words, and the "redirect not registered" mistake tells you exactly what to fix.
- The Admin page has buttons to fill in the redirect URL.

## 0.5.0

### 🔌 FordConnect 2.0
- CarTrak now uses Ford's current FordConnect 2.0 API: telemetry (the live vehicle data), your garage, health alerts, home charger, departure times and charge schedules. Version 0.4 and earlier used the older API, which is why connecting could fail.
- You only need your Ford client ID and secret now. No Application-Id.
- A new **Ford diagnostics** tool on the Admin page tests every Ford endpoint and can copy the answers with location and VIN removed, so problems are easy to report.
- An optional "Ford sign-in address" setting, in case Ford documents a different one.
- The older API is still available as the provider `fordconnect1`.
- The docs now spell out that everyone installing CarTrak registers their own Ford developer app; no Ford credentials are included or shared.

## 0.4.0

### 📡 Live and on the road
- A new **Live** tab: a full-screen dashboard of big tiles (charge, range, speed, charging power, this trip) that updates the moment a new reading arrives. It keeps the screen awake and has a Full screen button.
- **Road trip mode** asks Ford for readings more often while you drive, and turns itself off after 12 hours. If Ford asks CarTrak to slow down, it waits as long as Ford says and tells you.

### 📱 Phone, tablet and desktop
- Wider screens get a multi-column home page, multi-column lists, and a sidebar. Phones keep the single column.
- A per-device **Data use** setting (More): Auto, Normal, Live or **Data saver**. Data saver shows the essentials, refreshes every 5 minutes, and downloads the trip route and the full field list only when you tap. It only affects the device you set it on.
- Pages are compressed and refreshes with nothing new cost almost nothing.

### 📚 Docs
- A new guide for the live view, road trip mode and data saver.

## 0.3.1

### 🐛 Fixes
- Connect Ford now opens Ford's real sign-in page. Version 0.3.0 sent you to the wrong Ford address and showed "Not Found".

## 0.3.0

### ⚙️ Settings in the app
- A new Admin settings page (More -> Admin settings) for your Ford client ID, secret, Application-Id, redirect URL and time zone. Changes apply right away and the secret is never shown again. Settings in config.yaml or environment variables still work and win, and show as locked.
- Getting started and the home page send you to Admin settings until the Ford details are filled in.

### 🔒 Safer sign-in settings
- New settings for whether to trust proxy headers (on by default behind Cloudflare) and how many days a sign-in lasts.

### 📚 Docs
- The GitHub front page and a new field-by-field guide follow TrueNAS's Install Custom App form, with a table of every setting. The one-file YAML install is still available.

## 0.2.0

### 👋 Easier to start
- A Getting started guide opens the first time you sign in: connect Ford, install on your phone, add a work rule, add family. It stays available in More.
- The home page reminds you until setup is finished.

### 🔔 Versions and updates
- The version is always shown at the top of the screen and in More -> About.
- CarTrak tells you when a new version is out (you can turn this off) and how to install it. After an update, a note points you to what changed.
- A "What's new" page shows every release's changes inside the app.

### 🗂 Settings and storage
- Settings can live in a `config.yaml` file in a mapped `config` folder, next to the `data` folder, so both survive updates.
- TrueNAS: the YAML now maps both folders and downloads the newest release when you Stop then Start the app.
- A password recovery switch for the admin account.

### 📚 Docs
- Step-by-step guides on GitHub for TrueNAS, Cloudflare tunnels, Ford setup, settings, updating and troubleshooting.
- We now recommend a separate FordPass account for CarTrak, in the guides and in the Getting started page, so your main account is never at risk.

## 0.1.0

### 🚗 Your car's data
- Connects to Ford's FordConnect API: sign in once on Ford's own page, then CarTrak keeps itself updated.
- Dashboard cards for battery, charging, vehicle status, location and tires. A searchable "All data" page shows every field Ford returns.
- A built-in demo car lets you try everything without a Ford account.

### 🧭 Trips and charging
- Trips and charging sessions are found automatically, with distance, battery used and top speed.
- Label trips personal or work yourself, or let rules do it (for example weekdays 8am-5pm = work). Your own choice always wins.
- Work and personal mileage totals, and CSV export.

### 📱 On your phone
- Install it like an app from your phone's browser. Light and dark themes, in a Ford-inspired blue.
- Login with an admin account and read-only family accounts.

### 🛠 Running it
- Docker image, TrueNAS YAML and Cloudflare tunnel instructions.
