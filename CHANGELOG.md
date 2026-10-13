# Changelog

## 0.9.0

### 🩺 A Status screen that says what to do next
- A new Status page (More, then Status) checks the whole setup in plain language: Ford settings entered, Ford linked, last reading and how old it is, remote access, and whether an update is available. Each line says what is wrong and has a button to fix it.
- Home shows a short strip with the same result, so a problem is visible without opening any menu. Anyone signed in can see it; it never shows keys or passwords.

### 🚗 A dashboard built from the real car data
- New Doors and locks card: whether the car is locked and which doors, the tailgate or the hood are open.
- New Health card: warning lights the car reports and low tire pressure alerts, or "No warnings".
- Tire pressures and the lock state now read Ford's real layout (a list per wheel and per door), so the Tires card and the Locks line fill in with a linked car.
- The built-in demo car now uses the same layout, so these cards can be previewed without a car.

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
